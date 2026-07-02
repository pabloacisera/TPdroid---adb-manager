import { api } from './api.js';
import { i18n } from './i18n.js';
import { showStep, setStepState, renderProcessTable, renderAppsTable, showError, clearError, showToast, renderPagination, showLoading, hideLoading, showEmpty, hideEmpty, showLegend, showPermissionsModal, hidePermissionsModal, renderAdTable, showChannelsModal, hideChannelsModal, updateAppRow, renderUpdatePopover } from './ui.js';

const steps = ['prepare', 'connect', 'authorize', 'dashboard'];
let currentStep = 0;
let pollTimer = null;
let searchTimers = {};

// Estado del dispositivo conectado (capturado al autorizar)
let deviceSdkVersion = '';

const state = {
  processes: { data: [], page: 0, search: '', filterRunning: false, itemsPerPage: 25, loading: false },
  apps: { data: [], page: 0, search: '', filterNotif: 'all', itemsPerPage: 20, loading: false },
};

function goToStep(index) {
  steps.forEach((id, i) => {
    if (i < index) setStepState(id, 'completed');
    else if (i === index) setStepState(id, 'active');
    else setStepState(id, 'pending');
  });
  showStep(steps[index]);
  if (currentStep === 3 && index !== 3) {
    hideDisconnectBanner();
    stopHeartbeat();
  }
  currentStep = index;
}

function nextStep() {
  if (currentStep < steps.length - 1) {
    goToStep(currentStep + 1);
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

function pollUntil(apiFn, conditionFn, intervalMs, timeoutMs, onSuccess, onTimeout) {
  const start = Date.now();
  stopPolling();
  pollTimer = setInterval(async () => {
    if (Date.now() - start > timeoutMs) {
      stopPolling();
      onTimeout();
      return;
    }
    try {
      const data = await apiFn();
      if (conditionFn(data)) {
        stopPolling();
        onSuccess(data);
      }
  } catch (e) {
    console.warn('Version check failed (network or API error):', e);
  }
  }, intervalMs);
}

function startStep2() {
  clearError('connect');
  document.getElementById('step-connect-status').classList.remove('hidden');
  document.getElementById('step-connect-error').classList.add('hidden');
  pollUntil(
    () => api.getStatus(),
    (data) => data.connected === true,
    2000,
    60000,
    () => {
      document.getElementById('step-connect-status').classList.add('hidden');
      nextStep();
      startStep3();
    },
    () => {
      document.getElementById('step-connect-status').classList.add('hidden');
      document.getElementById('step-connect-error').classList.remove('hidden');
    }
  );
}

function startStep3() {
  clearError('authorize');
  document.getElementById('step-authorize-status').classList.remove('hidden');
  document.getElementById('step-authorize-error').classList.add('hidden');
  pollUntil(
    () => api.getDevice(),
    (data) => data.authorized === true && data.serial,
    2000,
    60000,
    (data) => {
      document.getElementById('step-authorize-status').classList.add('hidden');
      if (data && data.sdk_version) deviceSdkVersion = data.sdk_version;
      nextStep();
      loadDashboard();
    },
    () => {
      document.getElementById('step-authorize-status').classList.add('hidden');
      document.getElementById('step-authorize-error').classList.remove('hidden');
    }
  );
}

function filterData(data, searchTerm) {
  if (!searchTerm) return data;
  const q = searchTerm.toLowerCase();
  return data.filter(item => {
    const name = (item.name || item.package || '').toLowerCase();
    const label = (item.label || '').toLowerCase();
    return name.includes(q) || label.includes(q);
  });
}

function paginateData(data, page, itemsPerPage) {
  const start = page * itemsPerPage;
  return data.slice(start, start + itemsPerPage);
}

function filterAndRenderProcesses() {
  const s = state.processes;
  let filtered = filterData(s.data, s.search);
  if (s.filterRunning) {
    filtered = filtered.filter(p => p.status === 'R');
  }
  const totalPages = Math.max(1, Math.ceil(filtered.length / s.itemsPerPage));
  if (s.page >= totalPages) s.page = 0;
  const pageData = paginateData(filtered, s.page, s.itemsPerPage);

  hideEmpty('empty-processes');
  if (pageData.length === 0) {
    showEmpty('empty-processes');
  }
  renderProcessTable(pageData);
  renderPagination(
    'pagination-processes',
    s.page,
    totalPages,
    () => { s.page = Math.max(0, s.page - 1); filterAndRenderProcesses(); },
    () => { s.page = Math.min(totalPages - 1, s.page + 1); filterAndRenderProcesses(); }
  );
}

function filterAndRenderApps() {
  const s = state.apps;
  let filtered = filterData(s.data, s.search);
  if (s.filterNotif === 'on') {
    filtered = filtered.filter(a => !a.system_app && !a.notifications_disabled);
  } else if (s.filterNotif === 'off') {
    filtered = filtered.filter(a => !a.system_app && a.notifications_disabled);
  }
  const totalPages = Math.max(1, Math.ceil(filtered.length / s.itemsPerPage));
  if (s.page >= totalPages) s.page = 0;
  const pageData = paginateData(filtered, s.page, s.itemsPerPage);

  hideEmpty('empty-apps');
  if (pageData.length === 0) {
    showEmpty('empty-apps');
  }
  renderAppsTable(pageData);
  renderPagination(
    'pagination-apps',
    s.page,
    totalPages,
    () => { s.page = Math.max(0, s.page - 1); filterAndRenderApps(); },
    () => { s.page = Math.min(totalPages - 1, s.page + 1); filterAndRenderApps(); }
  );
}

async function loadProcesses() {
  const s = state.processes;
  if (s.loading) return;
  s.loading = true;
  s.page = 0;
  s.search = '';
  document.getElementById('search-processes').value = '';
  showLoading('loading-processes');
  hideEmpty('empty-processes');
  try {
    const data = await api.getProcesses();
    s.data = data;
    hideLoading('loading-processes');
    filterAndRenderProcesses();
    showLegend();
  } catch (e) {
    hideLoading('loading-processes');
    showToast(i18n.t('errors.generic'), 'error');
  }
  s.loading = false;
}

async function loadApps() {
  const s = state.apps;
  if (s.loading) return;
  s.loading = true;
  s.page = 0;
  s.search = '';
  document.getElementById('search-apps').value = '';
  showLoading('loading-apps');
  hideEmpty('empty-apps');
  try {
    const data = await api.getApps();
    s.data = data;
    hideLoading('loading-apps');
    filterAndRenderApps();
  } catch (e) {
    hideLoading('loading-apps');
    showToast(i18n.t('errors.generic'), 'error');
  }
  s.loading = false;
}

function loadDashboard() {
  loadProcesses();
  startHeartbeat();
}

let heartbeatTimer = null;
let disconnectCountdownTimer = null;
let disconnectStartTime = null;
const DISCONNECT_TIMEOUT = 5;

function showDisconnectBanner() {
  const banner = document.getElementById('disconnect-banner');
  banner.classList.remove('hidden');
  document.getElementById('disconnect-countdown').textContent = DISCONNECT_TIMEOUT;
  document.querySelector('.dashboard-scroll')?.classList.add('actions-disabled');
  disconnectStartTime = Date.now();
  disconnectCountdownTimer = setInterval(() => {
    const elapsed = Math.floor((Date.now() - disconnectStartTime) / 1000);
    const remaining = DISCONNECT_TIMEOUT - elapsed;
    if (remaining <= 0) {
      clearInterval(disconnectCountdownTimer);
      disconnectCountdownTimer = null;
      disconnectStartTime = null;
      hideDisconnectBanner();
      stopHeartbeat();
      goToStep(0);
    } else {
      document.getElementById('disconnect-countdown').textContent = remaining;
    }
  }, 1000);
}

function hideDisconnectBanner() {
  const banner = document.getElementById('disconnect-banner');
  banner.classList.add('hidden');
  document.querySelector('.dashboard-scroll')?.classList.remove('actions-disabled');
  if (disconnectCountdownTimer) {
    clearInterval(disconnectCountdownTimer);
    disconnectCountdownTimer = null;
  }
  disconnectStartTime = null;
}

function refreshCurrentTab() {
  const processesTab = document.getElementById('tab-processes-content');
  if (!processesTab.classList.contains('hidden')) {
    loadProcesses();
  } else {
    loadApps();
  }
}

async function checkConnection() {
  try {
    const status = await api.getStatus();
    if (status.connected) {
      if (disconnectStartTime !== null) {
        hideDisconnectBanner();
        refreshCurrentTab();
        showToast(i18n.t('device.reconnected'), 'success');
      }
    } else {
      if (disconnectStartTime === null) {
        showDisconnectBanner();
      }
    }
  } catch (e) {
    console.warn('Heartbeat check failed:', e);
  }
}

function startHeartbeat() {
  stopHeartbeat();
  heartbeatTimer = setInterval(checkConnection, 2000);
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
  if (disconnectCountdownTimer) {
    clearInterval(disconnectCountdownTimer);
    disconnectCountdownTimer = null;
  }
  disconnectStartTime = null;
}

document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    if (currentStep === 3) {
      stopHeartbeat();
    }
  } else {
    if (currentStep === 3) {
      startHeartbeat();
      checkConnection();
    }
  }
});

function setupSearch(inputId, tabState, renderFn) {
  document.getElementById(inputId).addEventListener('input', (e) => {
    const key = inputId + '_timer';
    clearTimeout(searchTimers[key]);
    searchTimers[key] = setTimeout(() => {
      tabState.search = e.target.value;
      tabState.page = 0;
      renderFn();
    }, 300);
  });
}

document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.dataset.action;
  const pkg = btn.dataset.package;

  if (action === 'force-stop') {
    btn.disabled = true;
    btn.textContent = '...';
    try {
      const res = await api.forceStop(pkg);
      if (res.success) {
        const msg = res.verified
          ? `${pkg} ${i18n.t('toast.stopped')}`
          : `${pkg} ${i18n.t('toast.maybeStillRunning')}`;
        showToast(msg, 'success');
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
      }
    } catch (e) {
      showToast(i18n.t('errors.generic'), 'error');
    }
    btn.disabled = false;
    setTimeout(() => loadProcesses(), 500);
  }

  if (action === 'disable-notif') {
    btn.disabled = true;
    btn.textContent = '...';
    const isGame = btn.dataset.isGame === 'true';
    try {
      const res = await api.disableNotification(pkg, isGame);
      if (res.success) {
        const msg = isGame
          ? `${pkg} ${i18n.t('toast.notificationsDisabled')} ${i18n.t('toast.gameLocked')}`
          : `${pkg} ${i18n.t('toast.notificationsDisabled')}`;
        showToast(msg, 'success');
        const app = state.apps.data.find(a => a.package === pkg);
        if (app) app.notifications_disabled = true;
        updateAppRow(pkg, true);
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
        btn.textContent = i18n.t('actions.disableNotif');
        btn.className = 'btn-action btn-notif-disable';
      }
    } catch (e) {
      showToast(i18n.t('errors.generic'), 'error');
      btn.textContent = i18n.t('actions.disableNotif');
      btn.className = 'btn-action btn-notif-disable';
    }
    btn.disabled = false;
  }

  if (action === 'enable-notif') {
    btn.disabled = true;
    btn.textContent = '...';
    const isGame = btn.dataset.isGame === 'true';
    try {
      const res = await api.enableNotification(pkg, isGame);
      if (res.success) {
        showToast(`${pkg} ${i18n.t('toast.notificationsEnabled')}`, 'success');
        const app = state.apps.data.find(a => a.package === pkg);
        if (app) app.notifications_disabled = false;
        updateAppRow(pkg, false);
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
        btn.textContent = i18n.t('actions.enableNotif');
        btn.className = 'btn-action btn-notif-enable';
      }
    } catch (e) {
      showToast(i18n.t('errors.generic'), 'error');
      btn.textContent = i18n.t('actions.enableNotif');
      btn.className = 'btn-action btn-notif-enable';
    }
    btn.disabled = false;
  }

  if (action === 'view-permissions') {
    const app = state.apps.data.find(a => a.package === pkg);
    if (app) showPermissionsModal(app.package, app.permissions);
  }

  if (action === 'view-channels') {
    let items = [];
    try { items = JSON.parse(btn.dataset.items.replace(/&quot;/g, '"')); } catch (_) {}
    showChannelsModal(pkg, items);
  }
});

function showUsernameModal() {
  document.getElementById('username-modal').classList.remove('hidden');
  document.getElementById('username-input').value = '';
  document.getElementById('username-error').classList.add('hidden');
  document.getElementById('username-input').focus();
}

function hideUsernameModal() {
  document.getElementById('username-modal').classList.add('hidden');
}

function saveUsername() {
  const name = document.getElementById('username-input').value.trim();
  if (!name) {
    document.getElementById('username-error').classList.remove('hidden');
    return;
  }
  localStorage.setItem('tpdroid_username', name);
  document.querySelector('.navbar-username').textContent = name;
  hideUsernameModal();
}

function loadUsername() {
  const saved = localStorage.getItem('tpdroid_username');
  if (saved) {
    document.querySelector('.navbar-username').textContent = saved;
  } else {
    showUsernameModal();
  }
}

document.getElementById('username-save').addEventListener('click', saveUsername);
document.getElementById('username-modal-close').addEventListener('click', hideUsernameModal);
document.getElementById('username-input').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') saveUsername();
});
document.getElementById('username-modal').addEventListener('click', (e) => {
  if (e.target === e.currentTarget) hideUsernameModal();
});

document.getElementById('perms-modal-close').addEventListener('click', hidePermissionsModal);
document.getElementById('perms-modal').addEventListener('click', (e) => {
  if (e.target === e.currentTarget) hidePermissionsModal();
});

document.getElementById('channels-modal-close').addEventListener('click', hideChannelsModal);
document.getElementById('channels-modal').addEventListener('click', (e) => {
  if (e.target === e.currentTarget) hideChannelsModal();
});

async function initApp() {
  await api.initSessionToken();
  try {
    const status = await api.getStatus();
    if (status.connected) {
      const device = await api.getDevice();
      if (device.authorized && device.serial) {
        if (device.sdk_version) deviceSdkVersion = device.sdk_version;
        goToStep(3);
        loadDashboard();
        return;
      }
      goToStep(2);
      startStep3();
      return;
    }
  } catch (e) {
    console.warn('initApp: backend not ready yet, showing connection screen:', e);
  }
  goToStep(0);
}

i18n.setLang('es');

document.getElementById('username-input').placeholder = i18n.t('username.placeholder');

initApp();

// Username modal — show after init if no stored username
setTimeout(loadUsername, 500);

document.getElementById('btn-prepare').addEventListener('click', () => {
  nextStep();
  startStep2();
});

document.getElementById('btn-retry-connect').addEventListener('click', startStep2);
document.getElementById('btn-retry-authorize').addEventListener('click', startStep3);

function switchTab(tab) {
  ['processes', 'apps', 'ads'].forEach(t => {
    const tabEl = document.getElementById(`tab-${t}`);
    const contentEl = document.getElementById(`tab-${t}-content`);
    if (!tabEl || !contentEl) return;
    tabEl.classList.toggle('active', t === tab);
    contentEl.classList.toggle('hidden', t !== tab);
  });

  if (tab === 'processes') {
    loadProcesses();
  } else if (tab === 'apps') {
    loadApps();
  } else if (tab === 'ads' && !adsState.scanned) {
    runAdScan();
  }
}

document.getElementById('tab-processes').addEventListener('click', () => switchTab('processes'));
document.getElementById('tab-apps').addEventListener('click', () => switchTab('apps'));
document.getElementById('tab-ads').addEventListener('click', () => switchTab('ads'));

const langEn = document.getElementById('lang-en');
const langEs = document.getElementById('lang-es');

function updateLangButtons(lang) {
  if (lang === 'en') {
    langEn.className = 'px-3 py-1 rounded text-sm font-medium lang-btn-active';
    langEs.className = 'px-3 py-1 rounded text-sm font-medium lang-btn-inactive';
  } else {
    langEs.className = 'px-3 py-1 rounded text-sm font-medium lang-btn-active';
    langEn.className = 'px-3 py-1 rounded text-sm font-medium lang-btn-inactive';
  }
}

langEn.addEventListener('click', () => {
  i18n.setLang('en');
  updateLangButtons('en');
  document.documentElement.lang = 'en';
});
langEs.addEventListener('click', () => {
  i18n.setLang('es');
  updateLangButtons('es');
  document.documentElement.lang = 'es';
});
updateLangButtons('es');

// ── Version check ────────────────────────────────────
let versionInfo = null;
let versionTimer = null;

async function checkVersion() {
  try {
    versionInfo = await api.getVersionInfo();
    renderUpdatePopover(versionInfo);
  } catch (e) {
    console.warn('Version check failed (network or API error):', e);
  }
}

function startVersionPolling() {
  stopVersionPolling();
  checkVersion();
  versionTimer = setInterval(checkVersion, 30 * 60 * 1000);
}

function stopVersionPolling() {
  if (versionTimer) {
    clearInterval(versionTimer);
    versionTimer = null;
  }
}

document.getElementById('update-bell').addEventListener('click', (e) => {
  e.stopPropagation();
  const popover = document.getElementById('update-popover');
  popover.classList.toggle('hidden');
  if (!popover.classList.contains('hidden') && versionInfo) {
    renderUpdatePopover(versionInfo);
  }
});

document.addEventListener('click', () => {
  const popover = document.getElementById('update-popover');
  if (!popover.classList.contains('hidden')) {
    popover.classList.add('hidden');
  }
});

document.getElementById('update-popover').addEventListener('click', (e) => {
  e.stopPropagation();
});

// Start version polling on init (runs even if not in dashboard)
startVersionPolling();

setupSearch('search-processes', state.processes, filterAndRenderProcesses);
setupSearch('search-apps', state.apps, filterAndRenderApps);

document.getElementById('refresh-processes').addEventListener('click', () => {
  const btn = document.getElementById('refresh-processes');
  btn.classList.add('spinning');
  loadProcesses();
  setTimeout(() => btn.classList.remove('spinning'), 600);
});

document.getElementById('refresh-apps').addEventListener('click', () => {
  const btn = document.getElementById('refresh-apps');
  btn.classList.add('spinning');
  loadApps();
  setTimeout(() => btn.classList.remove('spinning'), 600);
});

document.getElementById('filter-running').addEventListener('change', (e) => {
  state.processes.filterRunning = e.target.checked;
  state.processes.page = 0;
  filterAndRenderProcesses();
});

document.getElementById('filter-notif-status').addEventListener('change', (e) => {
  state.apps.filterNotif = e.target.value;
  state.apps.page = 0;
  filterAndRenderApps();
});

// === TAB: ADS & NOTIFICATION SCANNER ===

const adsState = {
  data: [],
  loading: false,
  scanned: false,
};

async function runAdScan() {
  if (adsState.loading) return;
  adsState.loading = true;
  adsState.scanned = true;
  showLoading('loading-ads');
  hideEmpty('empty-ads');
  try {
    const data = await api.scanAds();
    adsState.data = Array.isArray(data) ? data : [];
    hideLoading('loading-ads');
    if (adsState.data.length === 0) {
      showEmpty('empty-ads');
    }
    renderAdTable(adsState.data, () => deviceSdkVersion);
  } catch (err) {
    hideLoading('loading-ads');
    showToast(i18n.t('errors.generic'), 'error');
  }
  adsState.loading = false;
}

document.getElementById('scan-ads').addEventListener('click', runAdScan);


document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-action]');
  if (!btn) return;
  const action = btn.dataset.action;
  const pkg = btn.dataset.package;

  // --- Bloqueo por canal específico ---
  if (action === 'block-ad-channel') {
    let channels = [];
    try { channels = JSON.parse(btn.dataset.channels.replace(/&quot;/g, '"')); } catch (_) {}
    btn.disabled = true;
    btn.textContent = '...';
    try {
      const res = await api.blockAd(pkg, channels, deviceSdkVersion);
      if (res.success) {
        const label = res.full_blocked
          ? i18n.t('ads.fullBlockApplied')
          : i18n.t('ads.channelBlocked');
        showToast(`${pkg}: ${label}`, 'success');
        const entry = adsState.data.find(a => a.package === pkg);
        if (entry) {
          entry.blocked_channels = res.blocked_channels || [];
          entry.full_blocked = res.full_blocked || false;
          entry.notif_blocked = res.full_blocked || false;
        }
        renderAdTable(adsState.data, () => deviceSdkVersion);
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
        btn.disabled = false;
      }
    } catch (err) {
      showToast(i18n.t('errors.generic'), 'error');
      btn.disabled = false;
    }
  }

  // --- Bloqueo total del package ---
  if (action === 'block-ad-full') {
    const isBrowser = pkg.includes('chrome') || pkg.includes('firefox') ||
                      pkg.includes('browser') || pkg.includes('brave') ||
                      pkg.includes('opera') || pkg.includes('samsung');
    if (isBrowser) {
      const confirmed = window.confirm(i18n.t('ads.blockFullConfirm'));
      if (!confirmed) return;
    }
    btn.disabled = true;
    btn.textContent = '...';
    try {
      const res = await api.blockAdFull(pkg);
      if (res.success) {
        showToast(`${pkg}: ${i18n.t('ads.fullBlockApplied')}`, 'success');
        const entry = adsState.data.find(a => a.package === pkg);
        if (entry) {
          entry.full_blocked = true;
          entry.notif_blocked = true;
          entry.blocked_channels = [];
        }
        renderAdTable(adsState.data, () => deviceSdkVersion);
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
        btn.disabled = false;
      }
    } catch (err) {
      showToast(i18n.t('errors.generic'), 'error');
      btn.disabled = false;
    }
  }

  // --- Desbloqueo (simétrico) ---
  if (action === 'unblock-ad') {
    let bChannels = [];
    try { bChannels = JSON.parse(btn.dataset.blockedChannels.replace(/&quot;/g, '"')); } catch (_) {}
    btn.disabled = true;
    btn.textContent = '...';
    try {
      const res = await api.unblockAd(pkg, bChannels, deviceSdkVersion);
      if (res.success) {
        showToast(`${pkg}: ${i18n.t('ads.active')}`, 'success');
        const entry = adsState.data.find(a => a.package === pkg);
        if (entry) {
          entry.notif_blocked = false;
          entry.overlay_revoked = false;
          entry.blocked_channels = [];
          entry.full_blocked = false;
        }
        renderAdTable(adsState.data, () => deviceSdkVersion);
      } else {
        showToast(res.message || i18n.t('errors.generic'), 'error');
        btn.disabled = false;
      }
    } catch (err) {
      showToast(i18n.t('errors.generic'), 'error');
      btn.disabled = false;
    }
  }
});

// --- Botón: Bloqueo total de Chrome ---
document.getElementById('btn-block-chrome-total').addEventListener('click', async () => {
  const confirmed = window.confirm(i18n.t('ads.blockChromeTotalConfirm'));
  if (!confirmed) return;
  const btn = document.getElementById('btn-block-chrome-total');
  btn.disabled = true;
  const chromePackages = [
    'com.android.chrome', 'com.chrome.beta', 'com.chrome.dev',
    'com.chrome.canary', 'com.google.android.apps.chrome'
  ];
  let successCount = 0;
  let failCount = 0;
  for (const cpkg of chromePackages) {
    try {
      const res = await api.blockAdFull(cpkg);
      if (res.success) {
        successCount++;
        const entry = adsState.data.find(a => a.package === cpkg);
        if (entry) {
          entry.full_blocked = true;
          entry.notif_blocked = true;
          entry.blocked_channels = [];
        }
      } else {
        failCount++;
      }
    } catch (e) {
      failCount++;
    }
  }
  renderAdTable(adsState.data, () => deviceSdkVersion);
  btn.disabled = false;
  if (successCount > 0) {
    showToast(i18n.t('ads.blockChromeTotalDone', { n: successCount }), 'success');
  }
  if (failCount > 0) {
    showToast(`${failCount} ${i18n.t('errors.generic')}`, 'error');
  }
});

// --- Botón: Desactivar todas las notificaciones de apps ---
document.getElementById('btn-disable-all-apps-notif').addEventListener('click', async () => {
  const confirmed = window.confirm(i18n.t('actions.disableAllNotifConfirm'));
  if (!confirmed) return;
  const btn = document.getElementById('btn-disable-all-apps-notif');
  btn.disabled = true;
  const appsToDisable = state.apps.data.filter(a => !a.system_app && !a.notifications_disabled);
  let successCount = 0;
  let failCount = 0;
  for (const app of appsToDisable) {
    try {
      const res = await api.disableNotification(app.package, app.is_game);
      if (res.success) {
        successCount++;
        app.notifications_disabled = true;
      } else {
        failCount++;
      }
    } catch (e) {
      failCount++;
    }
  }
  filterAndRenderApps();
  btn.disabled = false;
  if (successCount > 0) {
    showToast(i18n.t('toast.disableAllDone', { n: successCount }), 'success');
  }
  if (failCount > 0) {
    showToast(i18n.t('toast.disableAllFailed', { n: failCount }), 'error');
  }
});
