const BASE = '/api';
let sessionToken = '';

export const api = {
  async initSessionToken() {
    try {
      const r = await fetch(`${BASE}/session-token`);
      const data = await r.json();
      sessionToken = data.token || '';
    } catch (e) {
      console.warn('Session token fetch failed, proceeding without:', e);
    }
  },
  async getStatus() {
    const r = await fetch(`${BASE}/status`);
    return r.json();
  },
  async getDevice() {
    const r = await fetch(`${BASE}/device`);
    return r.json();
  },
  async getProcesses() {
    const r = await fetch(`${BASE}/processes`);
    return r.json();
  },
  async forceStop(pkg) {
    const r = await fetch(`${BASE}/processes/force-stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg })
    });
    return r.json();
  },
  async getApps() {
    const r = await fetch(`${BASE}/apps`);
    return r.json();
  },
  async disableNotification(pkg, isGame = false) {
    const r = await fetch(`${BASE}/apps/disable-notification`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg, is_game: isGame })
    });
    return r.json();
  },
  async enableNotification(pkg, isGame = false) {
    const r = await fetch(`${BASE}/apps/enable-notification`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg, is_game: isGame })
    });
    return r.json();
  },
  async scanAds() {
    const r = await fetch(`${BASE}/ads/scan`);
    return r.json();
  },
  async blockAd(pkg, channels, sdkVersion) {
    const r = await fetch(`${BASE}/ads/block`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg, channels: channels || [], sdk_version: sdkVersion || '' })
    });
    return r.json();
  },
  async unblockAd(pkg, blockedChannels, sdkVersion) {
    const r = await fetch(`${BASE}/ads/unblock`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg, blocked_channels: blockedChannels || [], sdk_version: sdkVersion || '' })
    });
    return r.json();
  },
  async blockAdFull(pkg) {
    const r = await fetch(`${BASE}/ads/block-full`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Session-Token': sessionToken },
      body: JSON.stringify({ package: pkg })
    });
    return r.json();
  },
  async getVersionInfo() {
    const r = await fetch(`${BASE}/version`);
    return r.json();
  }
};
