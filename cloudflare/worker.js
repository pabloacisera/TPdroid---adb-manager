// Cloudflare Worker — Sistema de Licencias TPDroid
// Endpoints:
//   POST /activar    — Activa un código con un hw_id
//   POST /revalidar  — Revalida un archivo .lic existente

// Environment variables:
//   SUPABASE_URL        — https://<project>.supabase.co
//   SUPABASE_SERVICE_KEY — service_role key (admin)
//   LICENSE_SECRET      — secreto para HMAC

// ─── Rate limiting (in-memory sliding window) ──────────
// Nota: en producción con múltiples isolates esto es aproximado,
// pero sigue siendo una barrera efectiva contra fuerza bruta.
const rateLimitWindows = {
  '/activar':    { max: 5,  windowSec: 60 },
  '/revalidar':  { max: 30, windowSec: 60 },
  '/revocar':    { max: 10, windowSec: 60 },
  '/version':    { max: 60, windowSec: 60 },
  '/definitions': { max: 60, windowSec: 60 },
};

const rateLimitStore = new Map();

function cleanOldTimestamps(key, windowMs) {
  const now = Date.now();
  const timestamps = rateLimitStore.get(key) || [];
  const filtered = timestamps.filter(t => now - t < windowMs);
  if (filtered.length === 0) {
    rateLimitStore.delete(key);
  } else {
    rateLimitStore.set(key, filtered);
  }
  return filtered;
}

function checkRateLimit(path, clientIp) {
  const cfg = rateLimitWindows[path];
  if (!cfg) return true;
  const key = `${path}:${clientIp}`;
  const windowMs = cfg.windowSec * 1000;
  const recent = cleanOldTimestamps(key, windowMs);
  if (recent.length >= cfg.max) {
    return false;
  }
  recent.push(Date.now());
  rateLimitStore.set(key, recent);
  return true;
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const method = request.method;

    // CORS preflight
    if (method === 'OPTIONS') {
      return new Response(null, {
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type',
        },
      });
    }

    const clientIp = request.headers.get('CF-Connecting-IP') || request.headers.get('X-Forwarded-For') || 'unknown';

    // Rate limiting applies to all endpoints
    if (!checkRateLimit(url.pathname, clientIp)) {
      return jsonResponse({ error: 'Demasiadas solicitudes. Intente nuevamente en un minuto.' }, 429);
    }

    // Route by method + path
    if (method === 'GET') {
      switch (url.pathname) {
        case '/version':
          return handleGetVersion(env);
        case '/definitions':
          return handleGetDefinitions(env);
        default:
          return jsonResponse({ error: 'Not found' }, 404);
      }
    }

    if (method !== 'POST') {
      return jsonResponse({ error: 'Method not allowed' }, 405);
    }

    try {
      switch (url.pathname) {
        case '/activar':
          return handleActivar(request, env);
        case '/revalidar':
          return handleRevalidar(request, env);
        case '/revocar':
          return handleRevocar(request, env);
        default:
          return jsonResponse({ error: 'Not found' }, 404);
      }
    } catch (err) {
      return jsonResponse({ error: err.message }, 500);
    }
  },
};

// ─── Handlers ──────────────────────────────────────────

async function handleActivar(request, env) {
  const { codigo, hw_id, email } = await request.json();

  if (!codigo || !hw_id) {
    return jsonResponse({ error: 'codigo y hw_id son requeridos' }, 400);
  }

  // 1. Buscar código en Supabase
  const licencia = await queryLicencia(env, codigo);
  if (!licencia) {
    return jsonResponse({ error: 'Código de licencia inválido' }, 404);
  }

  if (licencia.revocado) {
    return jsonResponse({ error: 'Esta licencia ha sido revocada' }, 403);
  }

  if (licencia.usado) {
    return jsonResponse({ error: 'Código de licencia ya utilizado' }, 409);
  }

  // 2. Marcar como usado
  const now = new Date().toISOString();
  const updateFields = {
    usado: true,
    hw_id: hw_id,
    fecha_activacion: now,
  };
  if (email) {
    updateFields.email = email;
  }
  const updated = await updateLicencia(env, codigo, updateFields);

  if (!updated) {
    return jsonResponse({ error: 'Error al activar la licencia' }, 500);
  }

  // 3. Generar .lic (firmado con HMAC)
  const licFile = await buildLicFile(codigo, hw_id, now, env.LICENSE_SECRET);

  return jsonResponse({
    success: true,
    message: 'Licencia activada correctamente',
    lic: licFile,
  });
}

async function handleRevalidar(request, env) {
  const { lic, current_hw_id } = await request.json();

  if (!lic || !current_hw_id) {
    return jsonResponse({ error: 'lic y current_hw_id son requeridos' }, 400);
  }

  // 1. Validar HMAC
  const expectedHmac = await computeHmac(
    lic.codigo + lic.hw_id + lic.issued,
    env.LICENSE_SECRET
  );

  if (lic.hmac !== expectedHmac) {
    return jsonResponse({ error: 'Firma de licencia inválida' }, 401);
  }

  // 2. Validar que el hw_id coincida con el actual
  if (lic.hw_id !== current_hw_id) {
    return jsonResponse({ error: 'Esta licencia no corresponde a este equipo' }, 403);
  }

  // 3. Validar contra Supabase
  const dbLic = await queryLicencia(env, lic.codigo);
  if (!dbLic) {
    return jsonResponse({ error: 'Licencia no encontrada en el servidor' }, 404);
  }

  if (dbLic.revocado) {
    return jsonResponse({ error: 'Esta licencia ha sido revocada' }, 403);
  }

  if (dbLic.hw_id !== lic.hw_id) {
    return jsonResponse({ error: 'La licencia fue registrada con otro equipo' }, 403);
  }

  return jsonResponse({
    success: true,
    message: 'Licencia válida',
    hw_id: lic.hw_id,
    issued: lic.issued,
  });
}

async function handleRevocar(request, env) {
  // Requiere header de autenticación para proteger el endpoint
  const adminSecret = request.headers.get('X-Admin-Secret');
  if (!adminSecret || adminSecret !== env.ADMIN_SECRET) {
    return jsonResponse({ error: 'No autorizado' }, 401);
  }

  const { codigo } = await request.json();
  if (!codigo) {
    return jsonResponse({ error: 'codigo es requerido' }, 400);
  }

  const licencia = await queryLicencia(env, codigo);
  if (!licencia) {
    return jsonResponse({ error: 'Código no encontrado' }, 404);
  }

  if (licencia.revocado) {
    return jsonResponse({ error: 'La licencia ya estaba revocada' }, 409);
  }

  const ok = await updateLicencia(env, codigo, { revocado: true });
  if (!ok) {
    return jsonResponse({ error: 'Error al revocar la licencia' }, 500);
  }

  return jsonResponse({ success: true, message: `Licencia ${codigo} revocada` });
}

// ─── GET /version ──────────────────────────────────────

async function handleGetVersion(env) {
  const latest    = env.LATEST_VERSION || '0.2.0';
  const changelog = env.CHANGELOG      || 'Nueva actualización de aplicación disponible';
  const notesES   = env.NOTES_ES       || '';
  const notesEN   = env.NOTES_EN       || '';

  return jsonResponse({
    latest,
    download_url: 'https://github.com/pabloacisera/TPdroid---adb-manager/releases/latest',
    changelog,
    notes_es: notesES,
    notes_en: notesEN,
  });
}

// ─── GET /definitions ──────────────────────────────────

async function handleGetDefinitions(env) {
  return jsonResponse({
    known_non_game_prefixes: [
      "com.whatsapp", "com.facebook.katana", "com.facebook.orca",
      "com.facebook.lite", "com.facebook.mlite", "com.facebook.work",
      "com.instagram.android", "com.twitter.android", "com.linkedin.android",
      "com.snapchat.android", "com.zhiliaoapp.musically",
      "org.telegram.messenger", "org.telegram.plus", "org.telegram.bifrost",
      "com.discord", "com.slack", "com.reddit.frontpage",
      "com.pinterest", "com.tumblr", "com.viber.voip",
      "com.wechat", "com.linecorp", "com.kakao.talk",
      "com.kakao.story", "com.google.android.apps.messaging",
      "com.google.android.dialer", "com.google.android.apps.tachyon",
      "us.zoom.videomeetings", "com.skype.raider", "com.microsoft.teams",
      "com.verizon.messaging", "com.textra", "com.samsung.android.messaging",
      "com.android.mms", "com.android.dialer", "com.android.contacts",
      "com.simplemobiletools.smsmessenger", "com.signal",
      "com.android.chrome", "org.mozilla.firefox", "org.mozilla.firefox_beta",
      "org.mozilla.fenix", "org.mozilla.focus", "com.microsoft.emmx",
      "com.opera.browser", "com.opera.mini", "com.brave.browser",
      "com.sec.android.app.sbrowser", "com.duckduckgo.mobile.android",
      "com.vivaldi.browser", "com.kiwi.browser", "com.google.android.webview",
      "com.google.android.gm", "com.google.android.apps.maps",
      "com.google.android.apps.docs", "com.google.android.apps.photos",
      "com.google.android.apps.drive", "com.google.android.calendar",
      "com.google.android.keep", "com.google.android.apps.youtube",
      "com.google.android.youtube", "com.google.android.play.games",
      "com.google.android.googlequicksearchbox", "com.google.android.apps.news",
      "com.google.android.apps.translate", "com.google.android.apps.nbu.files",
      "com.google.android.apps.walletnfcrel",
      "com.google.android.apps.subscriptions.red",
      "com.google.android.apps.authenticator2",
      "com.google.android.apps.podcasts", "com.google.android.apps.youtube.music",
      "com.google.android.apps.magazines", "com.google.android.apps.fitness",
      "com.google.android.apps.nest", "com.google.android.apps.recorder",
      "com.google.android.apps.wellbeing", "com.google.android.apps.plus",
      "com.google.android.apps.tips", "com.google.android.gms",
      "com.google.android.gsf", "com.google.android.vending",
      "com.android.vending", "com.google.android.setupwizard",
      "com.google.android.syncadapters", "com.google.android.apps.books",
      "com.google.android.apps.cloudprint",
      "com.google.android.apps.documentation", "com.google.android.apps.mapslite",
      "com.google.android.apps.music", "com.google.android.apps.scholar",
      "com.google.android.apps.uploader", "com.google.android.backuptransport",
      "com.google.android.configupdater", "com.google.android.ext.services",
      "com.google.android.ext.shared", "com.google.android.gallery3d",
      "com.google.android.contacts", "com.google.android.deskclock",
      "com.google.android.launcher", "com.google.android.calculator",
      "com.google.android.gms", "com.google.android.gsf",
      "com.google.android.gsf.login", "com.google.android.syncadapters",
      "com.google.android.partnersetup", "com.google.android.feedback",
      "com.google.android.tag", "com.google.android.printservice.recommendation",
      "com.microsoft.office.word", "com.microsoft.office.excel",
      "com.microsoft.office.powerpoint", "com.microsoft.office.outlook",
      "com.microsoft.office.onenote", "com.microsoft.skydrive",
      "com.microsoft.teams", "com.microsoft.bing",
      "com.microsoft.rdc.android", "com.microsoft.office.officehub",
      "com.microsoft.office.office365", "com.microsoft.office.outlook",
      "com.microsoft.office.platform", "com.microsoft.office.storage",
      "com.microsoft.sharepoint", "com.microsoft.yammer",
      "com.microsoft.flow", "com.microsoft.powerapps",
      "com.adobe.reader", "com.adobe.psmobile",
      "com.adobe.lrmobile", "com.adobe.scan.android",
      "com.adobe.spark", "com.adobe.behance", "com.adobe.creativecloud",
      "com.adobe.phonegap", "com.adobe.premiereclip",
      "com.spotify.music", "com.netflix.mediaclient",
      "com.amazon.avod.thirdpartyclient", "com.disney.disneyplus",
      "com.hbo.hbonow", "com.hbo.max", "com.hulu.plus",
      "com.peacocktv.peacockandroid", "com.paramountplus",
      "com.apple.android.music", "com.deezer.android", "com.tidal.main",
      "com.soundcloud.android", "com.pandora.android", "com.shazam.android",
      "com.vlc.player", "com.mxtech.videoplayer",
      "com.amazon.mp3", "com.amazon.music",
      "com.youtube", "com.spotify", "com.plexapp.android",
      "com.plexapp.plex", "com.roku.web", "com.crunchyroll",
      "com.google.android.apps.youtube.music",
      "com.paypal.android.p2pmobile", "com.venmo",
      "com.squareup.cash", "com.chase.sig.android",
      "com.bankofamerica", "com.wf.wellsfargo",
      "us.hsbc.hsbcus", "com.citi.citimobile", "com.usaa.mobile",
      "com.americanexpress.android.acctsvcs", "com.capitalone.mobile",
      "net.firstdata.vfi", "com.walmart.wireless.citi",
      "com.scotiabank", "com.tdbank", "com.ally",
      "com.consolidated.tcfbank", "com.navyfederal",
      "com.vanguard", "com.fidelity.retail", "com.schwab.mobile",
      "com.robinhood", "com.coinbase.android",
      "com.binance", "com.kraken", "com.crypto.exchange",
      "com.bitcoin.app", "com.blockchain",
      "com.amazon.mShop.android", "com.amazon.mshop",
      "com.ebay.mobile", "com.walmart.android", "com.etsy.android",
      "com.alibaba.aliexpress", "com.shopee",
      "com.mercadopago", "com.mercadolibre",
      "com.shopify.shop", "com.ubercab", "com.lyft",
      "com.didiglobal", "com.grab", "com.gojek",
      "com.olx", "com.craigslist", "com.letgo",
      "com.wish", "com.alibaba.intl", "com.aliexpress",
      "com.rakuten", "com.target.android", "com.bestbuy",
      "com.home.depot", "com.lowes", "com.costco",
      "com.ubercab.eats", "com.dd.doordash", "com.grubhub.android",
      "com.seamless", "com.postmates", "com.justeat",
      "com.deliveroo", "com.takeaway", "com.hellofresh",
      "com.blueapron", "com.shipt",
      "com.evernote", "com.trello", "com.asana.app",
      "com.anydo", "com.todoist", "com.lastpass.lpandroid",
      "com.onelogin", "com.onepassword", "com.bitwarden",
      "com.box.android", "com.dropbox.android",
      "com.dropbox.paper", "com.eg.android.AlipayGphone",
      "com.weather", "com.weather.Weather",
      "com.teamviewer.teamviewer", "com.anydesk.android",
      "com.realvnc.viewer.android", "com.cloudflare.onedotonedotonedotone",
      "com.nordvpn.android", "com.expressvpn.vpn",
      "com.protonvpn.android", "com.cisco.anyconnect",
      "com.wireguard.android", "com.openvpn",
      "com.tailscale", "com.zerotier",
      "com.google.android.apps.fitness",
      "com.samsung.android.app.shealth", "com.myfitnesspal.android",
      "com.strava", "com.fitbit.FitbitMobile", "com.nike.plusgps",
      "com.getsomeheadspace.android", "com.calm.android",
      "com.underarmour.android", "com.endomondo.android",
      "com.runtastic.android", "com.mapmyrun",
      "com.zhiliaoapp.musically", "com.keek", "com.jefit",
      "com.loseit", "com.noom", "com.ww.weightwatchers",
      "com.azumio", "com.sleepcycle",
      "com.google.android.apps.maps", "com.waze",
      "com.ubercab", "com.lyft", "com.didiglobal",
      "com.airbnb.android", "com.booking", "com.expedia.bookings",
      "com.tripadvisor.tripadvisor", "com.opentable",
      "com.hertz", "com.avis", "com.sixrent",
      "com.hilton.android", "com.marriott.mrt", "com.ihg",
      "com.orbitz", "com.kayak.android", "com.skyscanner",
      "com.momondo", "com.rome2rio",
      "com.google.android.apps.nbu.files",
      "com.estrongs.android.pop", "com.pluto.solid.explorer",
      "com.maxmpz.audioplayer", "org.videolan.vlc",
      "com.speedtest.android", "org.wikipedia",
      "com.imdb.mobile", "com.duolingo",
      "com.grammarly.android", "com.quora.android",
      "com.medium.reader", "com.spotlight",
      "com.amazon.mShop.android", "com.amazon.avod.thirdpartyclient",
      "com.amazon.mp3", "com.amazon.kindle",
      "com.amazon.dee.app", "com.amazon.photos",
      "com.amazon.clouddrive.photos", "com.amazon.audio",
      "com.amazon.shh", "com.amazon.kindle",
      "com.sec.android.app.launcher", "com.samsung.android.app.shealth",
      "com.samsung.android.spay", "com.samsung.android.app.members",
      "com.samsung.android.app.notes", "com.sec.android.app.sbrowser",
      "com.samsung.android.calendar", "com.samsung.android.contacts",
      "com.samsung.android.messaging", "com.samsung.android.gallery",
      "com.samsung.android.game.gamehome", "com.samsung.android.samsungpass",
      "com.samsung.android.wallet", "com.samsung.android.bixby",
      "com.samsung.android.app.reminder", "com.samsung.android.app.contacts",
      "com.samsung.android.app.clock", "com.samsung.android.app.settings",
      "com.samsung.android.app.tips", "com.samsung.android.voc",
      "com.samsung.android.oneconnect", "com.samsung.android.scloud",
      "com.samsung.android.sdk", "com.samsung.android.kgclient",
      "com.samsung.android.providers", "com.samsung.android.themestore",
      "com.samsung.android.calendar", "com.samsung.android.weather",
      "com.samsung.android.video", "com.samsung.android.music",
      "com.samsung.android.app.watchmanager", "com.samsung.android.gear",
      "com.miui.", "com.xiaomi.", "com.mi.global.",
      "com.miui.securitycenter", "com.miui.cleanmaster",
      "com.miui.notes", "com.miui.gallery", "com.miui.player",
      "com.miui.browser", "com.miui.store", "com.miui.video",
      "com.miui.weather2", "com.miui.voiceassist", "com.miui.screenrecorder",
      "com.miui.compass", "com.miui.securityadd", "com.miui.personalassistant",
      "com.miui.calculator", "com.miui.backup", "com.miui.voiceassist",
      "com.miui.cloudbackup", "com.miui.cloudservice",
      "com.miui.virtualsim", "com.miui.weather",
      "com.miui.systemui", "com.miui.home", "com.miui.face",
      "com.huawei.", "com.android.huawei.",
      "com.huawei.android.", "com.huawei.systemmanager",
      "com.huawei.wallet", "com.huawei.health",
      "com.huawei.hwstartupguide", "com.huawei.hwid",
      "com.oneplus.", "com.oppo.", "com.vivo.",
      "com.realme.", "com.heytap.", "com.coloros.",
      "com.sonyericsson.", "com.sonymobile.", "com.lge.",
      "com.motorola.", "com.htc.", "com.nokia.",
      "com.bbm.", "com.blackberry.", "com.cyanogen.",
      "com.mediatek.", "com.qualcomm.", "com.tcl.",
      "com.zte.", "com.lenovo.", "com.asus.", "com.acer.",
      "com.google.android.apps.plus",
      "com.google.android.apps.walletnfcrel",
      "com.google.android.apps.maps",
      "com.google.android.apps.mapslite",
      "com.google.android.apps.books",
      "com.google.android.apps.cloudprint",
      "com.google.android.apps.documentation",
      "com.google.android.apps.fitness",
      "com.google.android.apps.magazines",
      "com.google.android.apps.music",
      "com.google.android.apps.news",
      "com.google.android.apps.podcasts",
      "com.google.android.apps.scholar",
      "com.google.android.apps.translate",
      "com.google.android.apps.uploader",
      "com.google.android.apps.youtube.music",
      "com.google.android.backuptransport",
      "com.google.android.calendar",
      "com.google.android.configupdater",
      "com.google.android.contacts",
      "com.google.android.deskclock",
      "com.google.android.gallery3d",
      "com.google.android.gm",
      "com.google.android.gms",
      "com.google.android.googlequicksearchbox",
      "com.google.android.gsf",
      "com.google.android.keep",
      "com.google.android.launcher",
      "com.google.android.partnersetup",
      "com.google.android.printservice.recommendation",
      "com.google.android.setupwizard",
      "com.google.android.syncadapters",
      "com.google.android.tag",
      "com.google.android.vending",
      "com.google.android.apps.walletnfcrel",
    ],
    ad_keywords: [
      "ad", "ads", "advert", "admob", "applovin", "ironsource", "chartboost",
      "vungle", "tapjoy", "mopub", "inmobi", "startapp", "airpush", "leadbolt",
      "show_ad", "display_ad", "push_notif", "promo", "offer", "banner",
      "interstitial", "rewarded", "mediation",
    ],
    game_segments: [
      ".game", ".games", ".gaming", "game.", "games.",
    ],
    game_engines: [
      "unity", "unreal", "cocos", "ironsource", "applovin",
      "vungle", "admob", "chartboost", "tapjoy",
    ],
  });
}

// ─── Supabase REST helpers ─────────────────────────────

function supabaseHeaders(env) {
  return {
    'Content-Type': 'application/json',
    'apikey': env.SUPABASE_SERVICE_KEY,
    'Authorization': `Bearer ${env.SUPABASE_SERVICE_KEY}`,
    'Prefer': 'return=representation',
  };
}

async function queryLicencia(env, codigo) {
  const url = `${env.SUPABASE_URL}/rest/v1/licencias?codigo=eq.${encodeURIComponent(codigo)}&limit=1`;
  const res = await fetch(url, {
    headers: supabaseHeaders(env),
  });
  if (!res.ok) return null;
  const data = await res.json();
  return data && data.length > 0 ? data[0] : null;
}

async function updateLicencia(env, codigo, fields) {
  const url = `${env.SUPABASE_URL}/rest/v1/licencias?codigo=eq.${encodeURIComponent(codigo)}`;
  const res = await fetch(url, {
    method: 'PATCH',
    headers: supabaseHeaders(env),
    body: JSON.stringify(fields),
  });
  return res.ok;
}

// ─── HMAC ──────────────────────────────────────────────

async function buildLicFile(codigo, hw_id, issued, secret) {
  const payload = codigo + hw_id + issued;
  const hmac = await computeHmac(payload, secret);
  return {
    codigo: codigo,
    hw_id: hw_id,
    issued: issued,
    hmac: hmac,
  };
}

async function computeHmac(payload, secret) {
  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(payload));
  return bytesToHex(new Uint8Array(signature));
}

function bytesToHex(bytes) {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

// ─── Response helper ───────────────────────────────────

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*',
    },
  });
}
