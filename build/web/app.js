// Tab panel switching
document.querySelectorAll('.tab[data-tab]').forEach(tab => {
  tab.addEventListener('click', () => {
    const target = tab.dataset.tab;
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
    const panel = document.getElementById('panel-' + target);
    if (panel) panel.classList.add('active');
  });
});

// ── Page texts ──────────────────────────────────────────────────────────────
//
// Every user-facing string comes from the active region (region.go), rendered
// into <script id="texts"> as JSON — no literals here.

const TEXT = JSON.parse(document.getElementById('texts').textContent);

// ── Open flow ───────────────────────────────────────────────────────────────
//
// Nothing to wire: a card is a plain link to /open/{slug}, which redirects to
// the app. No probes, no timers, no fallback popup.

// ── Info modal ──────────────────────────────────────────────────────────────

const modal = document.getElementById('appModal');

function openModal(btn) {
  // the icon ties the popup back to the card that opened it
  const icon = document.getElementById('appModalIcon');
  if (icon) {
    const src = btn.dataset.icon || '';
    if (src) { icon.src = src; icon.hidden = false; }
    else     { icon.removeAttribute('src'); icon.hidden = true; }
  }

  document.getElementById('appModalName').textContent = btn.dataset.name;
  document.getElementById('appModalDesc').textContent = btn.dataset.desc;

  const ul = document.getElementById('appModalFeatures');
  ul.innerHTML = '';
  const feats = (btn.dataset.features || '').split('|').map(s => s.trim()).filter(Boolean);
  feats.forEach(f => {
    const li = document.createElement('li');
    li.textContent = f;
    ul.appendChild(li);
  });
  ul.style.display = feats.length ? '' : 'none';

  modal.classList.add('open');
}
function closeModal() {
  if (modal) modal.classList.remove('open');
}

document.querySelectorAll('.app-info-btn').forEach(btn => {
  btn.addEventListener('click', () => openModal(btn));
});

if (modal) {
  modal.addEventListener('click', e => { if (e.target === modal) closeModal(); });
  const closeBtn = document.getElementById('appModalClose');
  if (closeBtn) closeBtn.addEventListener('click', closeModal);
}

// Esc closes the info popup
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeModal();
});

// ── Copy email to clipboard (info tab) ───────────────────────────────────────

const toast = document.getElementById('toast');
let toastTimer;
function showToast(msg) {
  if (!toast) return;
  toast.textContent = msg;
  toast.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove('show'), 2400);
}

async function copyText(text) {
  try {
    if (navigator.clipboard) { await navigator.clipboard.writeText(text); return true; }
  } catch { /* fall through to legacy path */ }
  try {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}

const copyEmailBtn = document.getElementById('copyEmail');
if (copyEmailBtn) {
  copyEmailBtn.addEventListener('click', async () => {
    const email = copyEmailBtn.dataset.email;
    const ok = await copyText(email);
    showToast(ok ? TEXT.emailCopied : email);
  });
}
