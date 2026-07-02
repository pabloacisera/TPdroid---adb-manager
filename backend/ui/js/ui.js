import { i18n } from './i18n.js';

export function showStep(stepId) {
  document.querySelectorAll('.step-content').forEach(el => {
    el.classList.add('hidden');
  });
  document.getElementById(`step-${stepId}`).classList.remove('hidden');
}

export function setStepState(stepId, state) {
  const indicators = document.querySelectorAll('.step-indicator');
  const stepIndex = ['prepare', 'connect', 'authorize', 'dashboard'].indexOf(stepId);
  indicators.forEach((el, i) => {
    if (i < stepIndex) el.dataset.state = 'completed';
    else if (i === stepIndex) el.dataset.state = state;
    else el.dataset.state = 'pending';
  });
}

export function renderProcessTable(processes) {
  const tbody = document.getElementById('processes-tbody');
  tbody.innerHTML = '';
  processes.forEach(p => {
    const tr = document.createElement('tr');
    if (p.system_process) tr.className = 'system-row';
    tr.innerHTML = `
      <td class="font-mono">${p.pid}</td>
      <td>
        <div class="process-name">
          ${p.system_process ? '<span class="lock-icon">🔒</span>' : ''}
          <span class="truncate min-w-0">${p.name}</span>
          ${p.system_process ? `<span class="badge-system">${i18n.t('badges.system')}</span>` : ''}
          ${p.status === 'R' ? '<span class="badge-running">RUN</span>' : ''}
        </div>
      </td>
      <td class="font-mono">${p.uid}</td>
      <td>${p.status}</td>
      <td>
        <button class="btn-action ${p.system_process ? 'btn-action-disabled' : 'btn-force-stop'}" data-action="force-stop" data-package="${p.name}" ${p.system_process ? 'disabled' : ''}>
          ${i18n.t('actions.forceStop')}
        </button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

export function renderAppsTable(apps) {
  const tbody = document.getElementById('apps-tbody');
  tbody.innerHTML = '';
  apps.forEach(a => {
    const tr = document.createElement('tr');
    if (a.system_app) tr.className = 'system-row';
    const disabled = a.notifications_disabled;
    const action = disabled ? 'enable-notif' : 'disable-notif';
    const btnClass = a.system_app ? 'btn-action-disabled' : (disabled ? 'btn-notif-enable' : 'btn-notif-disable');
    const label = disabled ? i18n.t('actions.enableNotif') : i18n.t('actions.disableNotif');

    const perms = a.permissions || [];
    let permsHtml;
    if (perms.length === 0) {
      permsHtml = '<span class="text-gray-500">' + i18n.t('table.noPermissions') + '</span>';
    } else {
      const maxVisible = 2;
      const visible = perms.slice(0, maxVisible);
      const hasMore = perms.length > maxVisible;
      permsHtml = '<span class="perms-truncate">' + visible.join(', ') + '</span>';
      if (hasMore) {
        permsHtml += '<button class="perms-view-btn" data-action="view-permissions" data-package="' + a.package + '">' + i18n.t('actions.viewPerms') + '</button>';
      }
    }

    let notifHtml;
    if (a.system_app) {
      notifHtml = `<span class="badge-system">${i18n.t('badges.system')}</span>`;
    } else if (disabled) {
      notifHtml = `<span class="notif-badge notif-badge-off">🔕 ${i18n.t('notif.off')}</span>`;
    } else {
      notifHtml = `<span class="notif-badge notif-badge-on">🔔 ${i18n.t('notif.on')}</span>`;
    }

    tr.innerHTML = `
      <td><div class="cell-wrap">${a.package}${a.is_game ? `<span class="badge-game" data-i18n="badges.game">${i18n.t('badges.game')}</span>` : ''}</div></td>
      <td><div class="cell-wrap">${a.label}</div></td>
      <td class="hidden sm:table-cell"><div class="cell-wrap">${notifHtml}</div></td>
      <td class="hidden lg:table-cell"><div class="cell-wrap text-xs text-gray-400">${permsHtml}</div></td>
      <td>
        <button class="btn-action ${btnClass}" data-action="${action}" data-package="${a.package}" data-is-game="${a.is_game}" ${a.system_app ? 'disabled' : ''}>
          ${label}
        </button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

export function showPermissionsModal(pkg, permissions) {
  const overlay = document.getElementById('perms-modal');
  document.getElementById('perms-modal-title').textContent = pkg;
  const body = document.getElementById('perms-modal-body');
  if (!permissions || permissions.length === 0) {
    body.innerHTML = '<div class="perm-empty">' + i18n.t('table.noPermissions') + '</div>';
  } else {
    body.innerHTML = permissions.map(p => '<div class="perm-item">' + p + '</div>').join('');
  }
  overlay.classList.remove('hidden');
}

export function hidePermissionsModal() {
  document.getElementById('perms-modal').classList.add('hidden');
}

export function showChannelsModal(pkg, items) {
  const overlay = document.getElementById('channels-modal');
  document.getElementById('channels-modal-title').textContent = `${pkg} — ${i18n.t('ads.channelsModalTitle')}`;
  const body = document.getElementById('channels-modal-body');
  if (!items || items.length === 0) {
    body.innerHTML = '<div class="perm-empty">—</div>';
  } else {
    body.innerHTML = items.map(item =>
      `<div class="channels-modal-item"><code>${item}</code></div>`
    ).join('');
  }
  overlay.classList.remove('hidden');
}

export function hideChannelsModal() {
  document.getElementById('channels-modal').classList.add('hidden');
}

export function showLegend() {
  document.getElementById('status-legend').classList.remove('hidden');
}

export function showError(stepId, message) {
  const step = document.getElementById(`step-${stepId}`);
  if (!step) return;
  let errorEl = step.querySelector('.step-error');
  if (!errorEl) {
    errorEl = document.createElement('div');
    errorEl.className = 'step-error mt-4 p-4 bg-red-900/50 border border-red-700 rounded-lg text-red-300 text-sm';
    step.querySelector('.bg-gray-800')?.appendChild(errorEl);
  }
  errorEl.textContent = message;
  errorEl.classList.remove('hidden');
}

export function clearError(stepId) {
  const step = document.getElementById(`step-${stepId}`);
  if (!step) return;
  const errorEl = step.querySelector('.step-error');
  if (errorEl) errorEl.classList.add('hidden');
}

export function showToast(message, type) {
  const container = document.getElementById('toast-container');
  const toast = document.createElement('div');
  toast.className = `toast toast-${type} transform translate-x-0`;
  toast.textContent = message;
  container.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = '0';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

export function renderPagination(containerId, currentPage, totalPages, onPrev, onNext) {
  const container = document.getElementById(containerId);
  container.innerHTML = '';
  if (totalPages <= 1) {
    container.classList.add('hidden');
    return;
  }
  container.classList.remove('hidden');
  const prevBtn = document.createElement('button');
  prevBtn.className = 'pagination-btn';
  prevBtn.textContent = i18n.t('actions.prev');
  prevBtn.disabled = currentPage === 0;
  prevBtn.addEventListener('click', onPrev);

  const info = document.createElement('span');
  info.className = 'pagination-info';
  info.textContent = `${currentPage + 1} / ${totalPages}`;

  const nextBtn = document.createElement('button');
  nextBtn.className = 'pagination-btn';
  nextBtn.textContent = i18n.t('actions.next');
  nextBtn.disabled = currentPage >= totalPages - 1;
  nextBtn.addEventListener('click', onNext);

  container.appendChild(prevBtn);
  container.appendChild(info);
  container.appendChild(nextBtn);
}

export function showLoading(containerId) {
  document.getElementById(containerId).classList.remove('hidden');
}

export function hideLoading(containerId) {
  document.getElementById(containerId).classList.add('hidden');
}

export function showEmpty(containerId) {
  document.getElementById(containerId).classList.remove('hidden');
}

export function hideEmpty(containerId) {
  document.getElementById(containerId).classList.add('hidden');
}

export function renderAdTable(entries, sdkVersionGetter) {
  const tbody = document.getElementById('ads-tbody');
  tbody.innerHTML = '';

  if (!entries || entries.length === 0) return;

  const getSdk = typeof sdkVersionGetter === 'function' ? sdkVersionGetter : () => '';

  entries.forEach(entry => {
    const tr = document.createElement('tr');
    const isChannelBlocked = entry.blocked_channels && entry.blocked_channels.length > 0;
    const isFullBlocked = entry.full_blocked || (entry.notif_blocked && !isChannelBlocked);
    const isAnyBlocked = isChannelBlocked || isFullBlocked;

    if (isAnyBlocked) tr.className = 'ads-blocked-row';

    const reasonsHtml = (entry.reasons || [])
      .map(r => `<span class="ads-badge-reason">${i18n.t('ads.reasons.' + r) || r}</span>`)
      .join('');

    const channels = entry.notif_channels || [];
    const alarms = entry.alarm_tags || [];
    const maxVisibleCh = 2;
    const maxVisibleAl = 1;
    let channelsHtml = '';

    if (channels.length === 0 && alarms.length === 0) {
      channelsHtml = '—';
    } else {
      const visibleChs = channels.slice(0, maxVisibleCh);
      const extraChs = channels.length - maxVisibleCh;
      const visibleAls = alarms.slice(0, maxVisibleAl);
      const extraAls = alarms.length - maxVisibleAl;
      const parts = [];

      visibleChs.forEach(c => {
        parts.push(`<code class="ads-channel channels-truncated" title="${c}">${c}</code>`);
      });
      visibleAls.forEach(t => {
        parts.push(`<code class="ads-alarm-tag channels-truncated" title="${t}">${t}</code>`);
      });

      const totalExtra = extraChs + extraAls;
      if (totalExtra > 0) {
        const allItems = [...channels, ...alarms];
        const allJson = JSON.stringify(allItems).replace(/"/g, '&quot;');
        parts.push(`<button class="channels-view-btn" data-action="view-channels" data-package="${entry.package}" data-items="${allJson}">+${totalExtra} ${i18n.t('ads.viewChannels')}</button>`);
      }

      channelsHtml = parts.join(' ');
    }

    let statusHtml;
    if (isChannelBlocked) {
      statusHtml = `<span class="ads-badge-channel-blocked">${i18n.t('ads.channelBlocked')}</span>`;
    } else if (isFullBlocked) {
      statusHtml = `<span class="ads-badge-blocked">${i18n.t('ads.blocked')}</span>`;
    } else {
      statusHtml = `<span class="ads-badge-active">${i18n.t('ads.active')}</span>`;
    }

    const sdkStr = getSdk();
    const sdk = parseInt(sdkStr, 10) || 0;
    const supportsChannels = sdk >= 26;
    const hasChannels = entry.notif_channels && entry.notif_channels.length > 0;

    let actionsHtml = '';
    if (entry.is_system_app) {
      actionsHtml = '<button class="btn-action btn-action-disabled" disabled>—</button>';
    } else if (isAnyBlocked) {
      const bChannels = entry.blocked_channels || [];
      const bChannelsJson = JSON.stringify(bChannels).replace(/"/g, '&quot;');
      actionsHtml = `
        <button class="btn-action btn-notif-enable"
          data-action="unblock-ad"
          data-package="${entry.package}"
          data-blocked-channels="${bChannelsJson}"
          data-full-blocked="${isFullBlocked}">
          ${i18n.t('ads.unblock')}
        </button>`;
    } else {
      if (supportsChannels && hasChannels) {
        const chJson = JSON.stringify(entry.notif_channels).replace(/"/g, '&quot;');
        actionsHtml = `
          <div class="ads-action-group">
            <button class="btn-action btn-force-stop"
              data-action="block-ad-channel"
              data-package="${entry.package}"
              data-channels="${chJson}"
              title="${i18n.t('ads.blockChannel')}">
              ${i18n.t('ads.blockChannel')}
            </button>
            <button class="btn-action btn-ads-full"
              data-action="block-ad-full"
              data-package="${entry.package}"
              title="${i18n.t('ads.blockFull')}">
              ☢
            </button>
          </div>`;
      } else {
        const title = !supportsChannels ? i18n.t('ads.sdkNoChannel') : i18n.t('ads.blockFull');
        actionsHtml = `
          <button class="btn-action btn-force-stop"
            data-action="block-ad-full"
            data-package="${entry.package}"
            title="${title}">
            ${i18n.t('ads.block')}
          </button>`;
      }
    }

    tr.innerHTML = `
      <td>
        <div class="cell-wrap">
          <span class="font-mono text-xs">${entry.package}</span>
          ${entry.is_system_app ? '<span class="badge-system">SYS</span>' : ''}
        </div>
      </td>
      <td><div class="cell-wrap flex flex-wrap gap-1">${reasonsHtml}</div></td>
      <td><div class="cell-wrap text-xs text-gray-400 flex flex-wrap gap-1">${channelsHtml}</div></td>
      <td>${statusHtml}</td>
      <td><div class="ads-action-cell">${actionsHtml}</div></td>
    `;
    tbody.appendChild(tr);
  });
}

export function renderUpdatePopover(info) {
  const body = document.getElementById('update-popover-body');
  const dot = document.getElementById('update-dot');
  const hasUpdate = info.latest && info.current && info.latest !== info.current;

  if (hasUpdate) {
    const lang = i18n.getLang();
    const notes = lang === 'es' ? info.notes_es : info.notes_en;
    const changelogText = notes || info.changelog;

    dot.classList.remove('hidden');
    body.innerHTML = `
      <div class="update-info-row">
        <span class="update-info-label">${i18n.t('update.current', { current: info.current })}</span>
        <span class="update-info-value">${info.current}</span>
      </div>
      <div class="update-info-row">
        <span class="update-info-label">${i18n.t('update.latest', { latest: info.latest })}</span>
        <span class="update-info-value"><span class="update-available-badge">${info.latest}</span></span>
      </div>
      ${info.download_url ? `<a href="${info.download_url}" class="update-download-btn" target="_blank" rel="noopener noreferrer">${i18n.t('update.download', { latest: info.latest })}</a>` : ''}
      ${changelogText ? `<div class="update-changelog"><strong>${i18n.t('update.changelog')}:</strong><br>${changelogText}</div>` : ''}
    `;
  } else {
    dot.classList.add('hidden');
    body.innerHTML = `<div class="update-no-update">${i18n.t('update.none')}</div>`;
  }
}

export function updateAppRow(pkg, notificationsDisabled) {
  const tbody = document.getElementById('apps-tbody');
  const rows = tbody.querySelectorAll('tr');
  for (const row of rows) {
    const actionBtn = row.querySelector(`button[data-package="${pkg}"]`);
    if (!actionBtn) continue;

    const newAction = notificationsDisabled ? 'enable-notif' : 'disable-notif';
    const newClass = notificationsDisabled ? 'btn-notif-enable' : 'btn-notif-disable';
    const newLabel = notificationsDisabled ? i18n.t('actions.enableNotif') : i18n.t('actions.disableNotif');
    actionBtn.dataset.action = newAction;
    actionBtn.className = `btn-action ${newClass}`;
    actionBtn.textContent = newLabel;
    actionBtn.disabled = false;

    const cells = row.querySelectorAll('td');
    if (cells.length >= 3) {
      const notifCell = cells[2];
      if (notificationsDisabled) {
        notifCell.innerHTML = `<div class="cell-wrap"><span class="notif-badge notif-badge-off">🔕 ${i18n.t('notif.off')}</span></div>`;
      } else {
        notifCell.innerHTML = `<div class="cell-wrap"><span class="notif-badge notif-badge-on">🔔 ${i18n.t('notif.on')}</span></div>`;
      }
    }
    break;
  }
}
