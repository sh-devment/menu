#!/usr/bin/env bash

set -euo pipefail

APP="menu"
APP_DIR="/opt/$APP"

BIN="$APP_DIR/$APP"
SERVICE="$APP_DIR/$APP.service"
ENV_FILE="$APP_DIR/$APP.env"
BACKUP="$APP_DIR/.backup"

echo "=== Deploy $APP ==="

# ── Validate ──────────────────────────────────────────────

test -f "bin/$APP"
test -f "build/$APP.service"

# Проверяем наличие конфигурации, не выводя значения.
test -n "${PORT:-}"
test -n "${AUTH_URL:-}"
test -n "${AUTH_INTERNAL:-}"
test -n "${APP_URL:-}"
test -n "${APP_TOKEN:-}"
test -n "${SECRET_KEY:-}"

# ── Backup current deployment ─────────────────────────────

rm -rf "$BACKUP"
mkdir -p "$BACKUP"

[ ! -f "$BIN" ] ||
  cp -a "$BIN" "$BACKUP/$APP"

[ ! -f "$SERVICE" ] ||
  cp -a "$SERVICE" "$BACKUP/$APP.service"

[ ! -f "$ENV_FILE" ] ||
  cp -a "$ENV_FILE" "$BACKUP/$APP.env"

# SQLite backup, если база уже существует.
#
# Бэкап сохраняем, но автоматически при rollback не
# восстанавливаем, чтобы случайно не потерять новые данные.
if [ -f "$APP_DIR/$APP.db" ]; then
  sqlite3 "$APP_DIR/$APP.db" \
    ".backup '$BACKUP/$APP.db'"
fi

# ── Environment ───────────────────────────────────────────

umask 077

cat > "$ENV_FILE.new" <<EOF
PORT=$PORT
AUTH_URL=$AUTH_URL
AUTH_INTERNAL=$AUTH_INTERNAL
APP_URL=$APP_URL
APP_TOKEN=$APP_TOKEN
SECRET_KEY=$SECRET_KEY
EOF

mv "$ENV_FILE.new" "$ENV_FILE"

# ── Binary ────────────────────────────────────────────────

install -m 0755 "bin/$APP" "$BIN.new"
mv "$BIN.new" "$BIN"

# ── Service ───────────────────────────────────────────────

install -m 0644 "build/$APP.service" "$SERVICE.new"
mv "$SERVICE.new" "$SERVICE"

# ── Restart ───────────────────────────────────────────────

sudo /usr/bin/systemctl daemon-reload
sudo /usr/bin/systemctl enable "$APP.service"
sudo /usr/bin/systemctl restart "$APP.service"

# ── Health check ──────────────────────────────────────────

OK=false

for i in $(seq 1 15); do
  if curl \
    --fail \
    --silent \
    --max-time 2 \
    "http://127.0.0.1:$PORT/" >/dev/null
  then
    OK=true
    break
  fi

  sleep 1
done

# ── Success ───────────────────────────────────────────────

if [ "$OK" = true ]; then
  sudo /usr/bin/systemctl is-active "$APP.service"

  echo
  echo "Version:"
  "$BIN" --version

  echo
  echo "=== Deploy successful ==="

  exit 0
fi

# ── Rollback ──────────────────────────────────────────────

echo
echo "=== Health check failed. Rolling back ==="

if [ -f "$BACKUP/$APP" ]; then
  cp -a "$BACKUP/$APP" "$BIN"
fi

if [ -f "$BACKUP/$APP.service" ]; then
  cp -a "$BACKUP/$APP.service" "$SERVICE"
fi

if [ -f "$BACKUP/$APP.env" ]; then
  cp -a "$BACKUP/$APP.env" "$ENV_FILE"
fi

# ВАЖНО:
#
# $BACKUP/$APP.db сохраняется для ручного восстановления.
# Автоматически SQLite DB здесь не откатываем.

sudo /usr/bin/systemctl daemon-reload

# На первом deploy предыдущей версии может вообще не быть.
if [ -f "$BACKUP/$APP" ]; then
  sudo /usr/bin/systemctl restart "$APP.service"

  echo "=== Previous application deployment restored ==="
else
  echo "=== First deployment failed; no previous version to restore ==="
fi

exit 1
