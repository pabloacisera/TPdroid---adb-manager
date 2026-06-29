package adb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func AdbBinaryPath() (string, error) {
	var rel string
	switch runtime.GOOS {
	case "windows":
		rel = filepath.Join("adb-binaries", "windows", "adb.exe")
	case "linux":
		rel = filepath.Join("adb-binaries", "linux", "adb")
	case "darwin":
		rel = filepath.Join("adb-binaries", "macos", "adb")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return rel, nil
}

func EnsureExecutable(path string) error {
	if runtime.GOOS != "windows" {
		return os.Chmod(path, 0755)
	}
	return nil
}

func runADB(adbPath string, args ...string) (string, error) {
	cmd := exec.Command(adbPath, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("adb error: %s | stderr: %s", err.Error(), stderr.String())
	}
	return out.String(), nil
}

func IsSystemProcess(uid string, packageName string) bool {
	systemUIDs := map[string]bool{
		"system": true,
		"root":   true,
		"shell":  true,
	}
	if systemUIDs[uid] {
		return true
	}
	browserAllowlist := map[string]bool{
		"com.android.chrome":  true,
		"com.android.browser": true,
	}
	if browserAllowlist[packageName] {
		return false
	}
	systemPrefixes := []string{"com.android.", "android.", "android"}
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(packageName, prefix) || packageName == "android" {
			return true
		}
	}
	return false
}

func GetDevices(adbPath string) ([]DeviceEntry, error) {
	out, err := runADB(adbPath, "devices")
	if err != nil {
		return nil, err
	}
	return parseDevices(out), nil
}

func parseDevices(output string) []DeviceEntry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var devices []DeviceEntry
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			devices = append(devices, DeviceEntry{
				Serial: parts[0],
				State:  parts[1],
			})
		}
	}
	return devices
}

func GetDeviceInfo(adbPath string, serial string) (DeviceInfo, error) {
	var info DeviceInfo
	info.Serial = serial

	brand, err := runADB(adbPath, "-s", serial, "shell", "getprop", "ro.product.brand")
	if err != nil {
		return info, err
	}
	info.Brand = strings.TrimSpace(brand)

	model, err := runADB(adbPath, "-s", serial, "shell", "getprop", "ro.product.model")
	if err != nil {
		return info, err
	}
	info.Model = strings.TrimSpace(model)

	version, err := runADB(adbPath, "-s", serial, "shell", "getprop", "ro.build.version.release")
	if err != nil {
		return info, err
	}
	info.AndroidVersion = strings.TrimSpace(version)

	sdk, err := runADB(adbPath, "-s", serial, "shell", "getprop", "ro.build.version.sdk")
	if err != nil {
		return info, err
	}
	info.SDKVersion = strings.TrimSpace(sdk)

	info.Authorized = true
	return info, nil
}

func GetProcesses(adbPath string, serial string) ([]Process, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "ps", "-A")
	if err != nil {
		return nil, err
	}
	return parseProcesses(out), nil
}

func VerifyProcessStopped(adbPath string, serial string, processName string) (bool, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "ps", "-A")
	if err != nil {
		return false, err
	}
	procs := parseProcesses(out)
	for _, p := range procs {
		if p.Name == processName {
			return false, nil
		}
	}
	return true, nil
}

func parseProcesses(output string) []Process {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var procs []Process
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		uid := fields[0]
		pid := fields[1]
		name := fields[8]
		status := fields[7]
		procs = append(procs, Process{
			PID:           pid,
			Name:          name,
			UID:           uid,
			Status:        status,
			SystemProcess: IsSystemProcess(uid, name),
		})
	}
	return procs
}

var knownNonGamePrefixes = []string{
	// Social & Communication
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

	// Browsers
	"com.android.chrome", "org.mozilla.firefox", "org.mozilla.firefox_beta",
	"org.mozilla.fenix", "org.mozilla.focus", "com.microsoft.emmx",
	"com.opera.browser", "com.opera.mini", "com.brave.browser",
	"com.sec.android.app.sbrowser", "com.duckduckgo.mobile.android",
	"com.vivaldi.browser", "com.kiwi.browser", "com.google.android.webview",

	// Google Apps
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

	// Google Play Services
	"com.google.android.gms", "com.google.android.gsf",
	"com.google.android.gsf.login", "com.google.android.syncadapters",
	"com.google.android.partnersetup", "com.google.android.feedback",
	"com.google.android.tag", "com.google.android.printservice.recommendation",

	// Microsoft Office & Services
	"com.microsoft.office.word", "com.microsoft.office.excel",
	"com.microsoft.office.powerpoint", "com.microsoft.office.outlook",
	"com.microsoft.office.onenote", "com.microsoft.skydrive",
	"com.microsoft.teams", "com.microsoft.bing",
	"com.microsoft.rdc.android", "com.microsoft.office.officehub",
	"com.microsoft.office.office365", "com.microsoft.office.outlook",
	"com.microsoft.office.platform", "com.microsoft.office.storage",
	"com.microsoft.sharepoint", "com.microsoft.yammer",
	"com.microsoft.flow", "com.microsoft.powerapps",

	// Adobe
	"com.adobe.reader", "com.adobe.psmobile",
	"com.adobe.lrmobile", "com.adobe.scan.android",
	"com.adobe.spark", "com.adobe.behance", "com.adobe.creativecloud",
	"com.adobe.phonegap", "com.adobe.premiereclip",

	// Music & Video Streaming
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

	// Banking & Finance
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

	// Shopping
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

	// Food Delivery
	"com.ubercab.eats", "com.dd.doordash", "com.grubhub.android",
	"com.seamless", "com.postmates", "com.justeat",
	"com.deliveroo", "com.takeaway", "com.hellofresh",
	"com.blueapron", "com.shipt",

	// Productivity
	"com.evernote", "com.trello", "com.asana.app",
	"com.anydo", "com.todoist", "com.lastpass.lpandroid",
	"com.onelogin", "com.onepassword", "com.bitwarden",
	"com.box.android", "com.dropbox.android",
	"com.dropbox.paper", "com.eg.android.AlipayGphone",
	"com.weather", "com.weather.Weather",

	// Remote Desktop & VPN
	"com.teamviewer.teamviewer", "com.anydesk.android",
	"com.realvnc.viewer.android", "com.cloudflare.onedotonedotonedotone",
	"com.nordvpn.android", "com.expressvpn.vpn",
	"com.protonvpn.android", "com.cisco.anyconnect",
	"com.wireguard.android", "com.openvpn",
	"com.tailscale", "com.zerotier",

	// Health & Fitness
	"com.google.android.apps.fitness",
	"com.samsung.android.app.shealth", "com.myfitnesspal.android",
	"com.strava", "com.fitbit.FitbitMobile", "com.nike.plusgps",
	"com.getsomeheadspace.android", "com.calm.android",
	"com.underarmour.android", "com.endomondo.android",
	"com.runtastic.android", "com.mapmyrun",
	"com.zhiliaoapp.musically", "com.keek", "com.jefit",
	"com.loseit", "com.noom", "com.ww.weightwatchers",
	"com.azumio", "com.sleepcycle",

	// Maps & Travel
	"com.google.android.apps.maps", "com.waze",
	"com.ubercab", "com.lyft", "com.didiglobal",
	"com.airbnb.android", "com.booking", "com.expedia.bookings",
	"com.tripadvisor.tripadvisor", "com.opentable",
	"com.hertz", "com.avis", "com.sixrent",
	"com.hilton.android", "com.marriott.mrt", "com.ihg",
	"com.orbitz", "com.kayak.android", "com.skyscanner",
	"com.momondo", "com.rome2rio",

	// Utilities & Tools
	"com.google.android.apps.nbu.files",
	"com.estrongs.android.pop", "com.pluto.solid.explorer",
	"com.maxmpz.audioplayer", "org.videolan.vlc",
	"com.speedtest.android", "org.wikipedia",
	"com.imdb.mobile", "com.duolingo",
	"com.grammarly.android", "com.quora.android",
	"com.medium.reader", "com.spotlight",

	// Amazon
	"com.amazon.mShop.android", "com.amazon.avod.thirdpartyclient",
	"com.amazon.mp3", "com.amazon.kindle",
	"com.amazon.dee.app", "com.amazon.photos",
	"com.amazon.clouddrive.photos", "com.amazon.audio",
	"com.amazon.shh", "com.amazon.kindle",

	// Samsung
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

	// Xiaomi (MIUI)
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

	// Huawei
	"com.huawei.", "com.android.huawei.",
	"com.huawei.android.", "com.huawei.systemmanager",
	"com.huawei.wallet", "com.huawei.health",
	"com.huawei.hwstartupguide", "com.huawei.hwid",

	// OnePlus / Oppo / Vivo / Realme
	"com.oneplus.", "com.oppo.", "com.vivo.",
	"com.realme.", "com.heytap.", "com.coloros.",

	// Other OEMs
	"com.sonyericsson.", "com.sonymobile.", "com.lge.",
	"com.motorola.", "com.htc.", "com.nokia.",
	"com.bbm.", "com.blackberry.", "com.cyanogen.",
	"com.mediatek.", "com.qualcomm.", "com.tcl.",
	"com.zte.", "com.lenovo.", "com.asus.", "com.acer.",

	// Non-game apps with VIBRATE+INTERNET that may be misdetected
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
}

func isKnownNonGame(pkg string) bool {
	p := strings.ToLower(pkg)
	for _, prefix := range knownNonGamePrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func IsGameApp(pkg string, permissions []string) bool {
	p := strings.ToLower(pkg)

	if isKnownNonGame(p) {
		return false
	}

	gameSegments := []string{
		".game", ".games", ".gaming", "game.", "games.",
	}
	for _, seg := range gameSegments {
		if strings.Contains(p, seg) {
			return true
		}
	}

	if strings.HasSuffix(p, "game") || strings.HasSuffix(p, "games") {
		return true
	}

	gameEngines := []string{
		"unity", "unreal", "cocos", "ironsource", "applovin",
		"vungle", "admob", "chartboost", "tapjoy",
	}
	for _, eng := range gameEngines {
		if strings.Contains(p, eng) {
			return true
		}
	}

	return false
}

func ForceStop(adbPath string, serial string, pkg string) error {
	_, err := runADB(adbPath, "-s", serial, "shell", "am", "force-stop", pkg)
	return err
}

func GetApps(adbPath string, serial string) ([]App, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "pm", "list", "packages", "-3")
	if err != nil {
		return nil, err
	}
	pkgs := parsePackages(out)

	type pkgResult struct {
		perms                 []string
		notificationsDisabled bool
		isGame                bool
	}

	results := make([]pkgResult, len(pkgs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for i, pkg := range pkgs {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			perms, err := GetAppPermissions(adbPath, serial, p)
			if err != nil {
				perms = []string{}
			}
			notifDisabled, _ := GetNotificationStatus(adbPath, serial, p)
			results[idx] = pkgResult{perms: perms, notificationsDisabled: notifDisabled, isGame: IsGameApp(p, perms)}
		}(i, pkg)
	}

	wg.Wait()

	apps := make([]App, len(pkgs))
	for i, pkg := range pkgs {
		apps[i] = App{
			Package:               pkg,
			Label:                 pkg,
			SystemApp:             IsSystemProcess("", pkg),
			IsGame:                results[i].isGame,
			Permissions:           results[i].perms,
			NotificationsDisabled: results[i].notificationsDisabled,
		}
	}
	return apps, nil
}

func parsePackages(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var pkgs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkgs = append(pkgs, strings.TrimPrefix(line, "package:"))
		}
	}
	return pkgs
}

func GetAppPermissions(adbPath string, serial string, pkg string) ([]string, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "dumpsys", "package", pkg)
	if err != nil {
		return nil, err
	}
	return parsePermissions(out), nil
}

func parsePermissions(output string) []string {
	var perms []string
	lines := strings.Split(output, "\n")
	inPermSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "install permissions:") ||
			strings.Contains(trimmed, "runtime permissions:") {
			inPermSection = true
			continue
		}
		if inPermSection {
			if strings.HasPrefix(trimmed, "android.permission.") ||
				strings.HasPrefix(trimmed, "com.android.") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) > 0 {
					perms = append(perms, strings.TrimSpace(parts[0]))
				}
			} else if trimmed == "" || strings.HasSuffix(trimmed, ":") {
				inPermSection = false
			}
		}
	}
	return perms
}

func GetNotificationStatus(adbPath string, serial string, pkg string) (bool, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "appops", "get", pkg, "POST_NOTIFICATION")
	if err != nil {
		return false, err
	}
	// On SDK < 33: "mode=deny". On SDK 33+: "ignore" after pm revoke.
	trimmed := strings.TrimSpace(out)
	if strings.Contains(trimmed, "deny") || strings.Contains(trimmed, "ignore") {
		return true, nil
	}
	return false, nil
}

func SetNotification(adbPath string, serial string, pkg string, mode string) error {
	_, err := runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "POST_NOTIFICATION", mode)
	return err
}

func DisableNotifications(adbPath string, serial string, pkg string) error {
	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "POST_NOTIFICATION", "deny")
	runADB(adbPath, "-s", serial, "shell", "pm", "revoke", pkg, "android.permission.POST_NOTIFICATIONS")
	runADB(adbPath, "-s", serial, "shell", "pm", "set-permission-flags", pkg, "android.permission.POST_NOTIFICATIONS", "user-set")
	runADB(adbPath, "-s", serial, "shell", "pm", "clear-permission-flags", pkg, "android.permission.POST_NOTIFICATIONS", "user-fixed")
	return nil
}

func EnableNotifications(adbPath string, serial string, pkg string) error {
	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "POST_NOTIFICATION", "allow")
	runADB(adbPath, "-s", serial, "shell", "pm", "grant", pkg, "android.permission.POST_NOTIFICATIONS")
	runADB(adbPath, "-s", serial, "shell", "pm", "set-permission-flags", pkg, "android.permission.POST_NOTIFICATIONS", "user-set")
	runADB(adbPath, "-s", serial, "shell", "pm", "clear-permission-flags", pkg, "android.permission.POST_NOTIFICATIONS", "user-fixed")
	return nil
}

// GetActiveNotifications parsea dumpsys notification y devuelve un mapa de package → canales activos.
func GetActiveNotifications(adbPath string, serial string) (map[string][]string, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "dumpsys", "notification", "--noredact")
	if err != nil {
		out, err = runADB(adbPath, "-s", serial, "shell", "dumpsys", "notification")
		if err != nil {
			return nil, fmt.Errorf("dumpsys notification failed: %w", err)
		}
	}
	return parseActiveNotifications(out), nil
}

func parseActiveNotifications(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	var currentPkg string
	var currentChannel string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "NotificationRecord(") {
			currentPkg = ""
			currentChannel = ""
			if idx := strings.Index(trimmed, "pkg="); idx != -1 {
				rest := trimmed[idx+4:]
				parts := strings.Fields(rest)
				if len(parts) > 0 {
					currentPkg = strings.TrimRight(parts[0], ",)")
				}
			}
			continue
		}

		if currentPkg != "" && strings.Contains(trimmed, "NotificationChannel{") {
			if idx := strings.Index(trimmed, "mId='"); idx != -1 {
				rest := trimmed[idx+5:]
				if end := strings.Index(rest, "'"); end != -1 {
					currentChannel = rest[:end]
				}
			}
			if currentChannel == "" {
				if idx := strings.Index(trimmed, "mName="); idx != -1 {
					rest := trimmed[idx+6:]
					parts := strings.SplitN(rest, ",", 2)
					currentChannel = strings.TrimSpace(parts[0])
				}
			}
			if currentPkg != "" && currentChannel != "" {
				channelKey := currentChannel
				found := false
				for _, ch := range result[currentPkg] {
					if ch == channelKey {
						found = true
						break
					}
				}
				if !found {
					result[currentPkg] = append(result[currentPkg], channelKey)
				}
			}
			continue
		}

		if currentPkg == "" && strings.HasPrefix(trimmed, "pkg=") {
			currentPkg = strings.TrimPrefix(trimmed, "pkg=")
			currentPkg = strings.TrimSpace(currentPkg)
		}
	}
	return result
}

var adKeywords = []string{
	"ad", "ads", "advert", "admob", "applovin", "ironsource", "chartboost",
	"vungle", "tapjoy", "mopub", "inmobi", "startapp", "airpush", "leadbolt",
	"show_ad", "display_ad", "push_notif", "promo", "offer", "banner",
	"interstitial", "rewarded", "mediation",
}

// GetAdAlarms parsea dumpsys alarm y retorna mapa de package → tags sospechosos.
func GetAdAlarms(adbPath string, serial string) (map[string][]string, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "dumpsys", "alarm")
	if err != nil {
		return nil, fmt.Errorf("dumpsys alarm failed: %w", err)
	}
	return parseAdAlarms(out), nil
}

func parseAdAlarms(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	var currentPkg string
	var currentTag string
	var hasRepeat bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "pkg=") && (strings.Contains(trimmed, "Alarm{") || strings.HasPrefix(trimmed, "ALARM ")) {
			currentPkg = ""
			currentTag = ""
			hasRepeat = false
			if idx := strings.Index(trimmed, "pkg="); idx != -1 {
				rest := trimmed[idx+4:]
				parts := strings.Fields(rest)
				if len(parts) > 0 {
					currentPkg = strings.Trim(parts[0], ",}")
				}
			}
			continue
		}

		if currentPkg != "" && strings.HasPrefix(trimmed, "tag=") {
			currentTag = strings.TrimPrefix(trimmed, "tag=")
			currentTag = strings.TrimSpace(currentTag)
			continue
		}

		if currentPkg != "" && strings.HasPrefix(trimmed, "repeatInterval=") {
			val := strings.TrimPrefix(trimmed, "repeatInterval=")
			if val != "0" && val != "" {
				hasRepeat = true
			}
			continue
		}

		if trimmed == "" && currentPkg != "" {
			if shouldFlagAlarm(currentPkg, currentTag, hasRepeat) {
				tagToStore := currentTag
				if tagToStore == "" {
					tagToStore = "alarma_repetitiva"
				}
				found := false
				for _, t := range result[currentPkg] {
					if t == tagToStore {
						found = true
						break
					}
				}
				if !found {
					result[currentPkg] = append(result[currentPkg], tagToStore)
				}
			}
			currentPkg = ""
			currentTag = ""
			hasRepeat = false
		}
	}
	return result
}

func shouldFlagAlarm(pkg string, tag string, hasRepeat bool) bool {
	if IsSystemProcess("", pkg) {
		return false
	}
	tagLower := strings.ToLower(tag)
	for _, kw := range adKeywords {
		if strings.Contains(tagLower, kw) {
			return true
		}
	}
	if hasRepeat && !isKnownNonGame(pkg) {
		return true
	}
	return false
}

// GetActiveOverlays parsea dumpsys window windows y retorna packages con overlays activos.
func GetActiveOverlays(adbPath string, serial string) ([]string, error) {
	out, err := runADB(adbPath, "-s", serial, "shell", "dumpsys", "window", "windows")
	if err != nil {
		return nil, fmt.Errorf("dumpsys window windows failed: %w", err)
	}
	return parseActiveOverlays(out), nil
}

func parseActiveOverlays(output string) []string {
	var pkgs []string
	seen := make(map[string]bool)
	lines := strings.Split(output, "\n")

	var currentPkg string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "TYPE_APPLICATION_OVERLAY") || strings.Contains(trimmed, "type=2038") {
			if currentPkg != "" && !IsSystemProcess("", currentPkg) && !seen[currentPkg] {
				pkgs = append(pkgs, currentPkg)
				seen[currentPkg] = true
			}
			continue
		}

		if strings.Contains(trimmed, "Window{") {
			currentPkg = ""
			parts := strings.Fields(trimmed)
			for _, p := range parts {
				if strings.Contains(p, ".") && !strings.HasPrefix(p, "0x") {
					pkg := strings.SplitN(p, "/", 2)[0]
					pkg = strings.Trim(pkg, "}")
					if strings.Count(pkg, ".") >= 1 && !strings.Contains(pkg, "=") {
						currentPkg = pkg
						break
					}
				}
			}
		}
	}
	return pkgs
}

// ScanAdSources ejecuta los tres dumpsys en paralelo y devuelve la lista unificada.
func ScanAdSources(adbPath string, serial string) ([]AdEntry, error) {
	type notifResult struct {
		data map[string][]string
		err  error
	}
	type alarmResult struct {
		data map[string][]string
		err  error
	}
	type overlayResult struct {
		data []string
		err  error
	}

	notifCh := make(chan notifResult, 1)
	alarmCh := make(chan alarmResult, 1)
	overlayCh := make(chan overlayResult, 1)

	go func() {
		d, e := GetActiveNotifications(adbPath, serial)
		notifCh <- notifResult{d, e}
	}()
	go func() {
		d, e := GetAdAlarms(adbPath, serial)
		alarmCh <- alarmResult{d, e}
	}()
	go func() {
		d, e := GetActiveOverlays(adbPath, serial)
		overlayCh <- overlayResult{d, e}
	}()

	nr := <-notifCh
	ar := <-alarmCh
	or := <-overlayCh

	notifMap := nr.data
	if notifMap == nil {
		notifMap = make(map[string][]string)
	}
	alarmMap := ar.data
	if alarmMap == nil {
		alarmMap = make(map[string][]string)
	}
	overlaySet := make(map[string]bool)
	for _, pkg := range or.data {
		overlaySet[pkg] = true
	}

	allPkgs := make(map[string]bool)
	for pkg := range notifMap {
		allPkgs[pkg] = true
	}
	for pkg := range alarmMap {
		allPkgs[pkg] = true
	}
	for pkg := range overlaySet {
		allPkgs[pkg] = true
	}

	var entries []AdEntry
	for pkg := range allPkgs {
		entry := AdEntry{
			Package:     pkg,
			IsSystemApp: IsSystemProcess("", pkg),
		}

		if channels, ok := notifMap[pkg]; ok && len(channels) > 0 {
			entry.NotifChannels = channels
			isWebPush := false
			for _, ch := range channels {
				chLower := strings.ToLower(ch)
				if strings.Contains(chLower, "browser") || strings.Contains(chLower, "web") ||
					strings.Contains(chLower, "push") || strings.Contains(chLower, "site") {
					isWebPush = true
					break
				}
			}
			if isWebPush {
				entry.Reasons = append(entry.Reasons, "web_push")
			} else {
				entry.Reasons = append(entry.Reasons, "active_notif")
			}
		}

		if tags, ok := alarmMap[pkg]; ok && len(tags) > 0 {
			entry.AlarmTags = tags
			entry.Reasons = append(entry.Reasons, "ad_alarm")
		}

		if overlaySet[pkg] {
			entry.Reasons = append(entry.Reasons, "overlay_window")
		}

		notifBlocked, _ := GetNotificationStatus(adbPath, serial, pkg)
		entry.NotifBlocked = notifBlocked

		opsOut, err := runADB(adbPath, "-s", serial, "shell", "appops", "get", pkg, "SYSTEM_ALERT_WINDOW")
		if err == nil && strings.Contains(opsOut, "deny") {
			entry.OverlayRevoked = true
		}

		if len(entry.Reasons) > 0 {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// BlockAdSource bloquea definitivamente una fuente de ads/notificaciones intrusivas.
func BlockAdSource(adbPath string, serial string, pkg string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot block system app: %s", pkg)
	}

	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "SYSTEM_ALERT_WINDOW", "deny")

	if err := DisableNotifications(adbPath, serial, pkg); err != nil {
		return fmt.Errorf("failed to disable notifications for %s: %w", pkg, err)
	}

	runADB(adbPath, "-s", serial, "shell", "am", "force-stop", pkg)

	return nil
}

// UnblockAdSource restaura una app bloqueada por BlockAdSource.
func UnblockAdSource(adbPath string, serial string, pkg string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot unblock system app: %s", pkg)
	}

	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "SYSTEM_ALERT_WINDOW", "allow")

	return EnableNotifications(adbPath, serial, pkg)
}

// BlockNotifChannel bloquea un canal de notificación específico de un package.
// Requiere Android 8+ (SDK >= 26). Usa "cmd notification disable-channel".
// Retorna error si el comando falla o no está disponible.
func BlockNotifChannel(adbPath string, serial string, pkg string, channelId string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot block channel for system app: %s", pkg)
	}
	out, err := runADB(adbPath, "-s", serial, "shell",
		"cmd", "notification", "disable-channel", pkg, channelId)
	if err != nil {
		return fmt.Errorf("disable-channel failed for %s/%s: %w | out: %s", pkg, channelId, err, out)
	}
	return nil
}

// UnblockNotifChannel reactiva un canal de notificación específico.
func UnblockNotifChannel(adbPath string, serial string, pkg string, channelId string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot unblock channel for system app: %s", pkg)
	}
	out, err := runADB(adbPath, "-s", serial, "shell",
		"cmd", "notification", "enable-channel", pkg, channelId)
	if err != nil {
		return fmt.Errorf("enable-channel failed for %s/%s: %w | out: %s", pkg, channelId, err, out)
	}
	return nil
}

func parseSDK(sdkVersion string) int {
	sdkVersion = strings.TrimSpace(sdkVersion)
	sdk := 0
	for _, ch := range sdkVersion {
		if ch >= '0' && ch <= '9' {
			sdk = sdk*10 + int(ch-'0')
		} else {
			break
		}
	}
	return sdk
}

// SupportsChannelBlock verifica si el dispositivo soporta bloqueo por canal.
// Requiere SDK >= 26 (Android 8.0+) y < 33 (cmd notification disable-channel
// fue removido en Android 13).
func SupportsChannelBlock(sdkVersion string) bool {
	sdk := parseSDK(sdkVersion)
	return sdk >= 26 && sdk < 33
}

// BlockAdSourceSmart bloquea una fuente intrusiva de forma inteligente:
// - Si SDK ∈ [26, 33) Y hay canales identificados: bloquea SOLO esos canales
// - Si SDK < 26 O SDK ≥ 33 O no hay canales: bloquea el package completo
// - Siempre revoca el overlay si lo tenía
// Retorna los canales bloqueados (puede ser nil si fue bloqueo total).
func BlockAdSourceSmart(adbPath string, serial string, pkg string, channels []string, sdkVersion string) (blockedChannels []string, fullBlocked bool, err error) {
	if IsSystemProcess("", pkg) {
		return nil, false, fmt.Errorf("cannot block system app: %s", pkg)
	}

	// Siempre revocar overlay (no-op si no lo tenía)
	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "SYSTEM_ALERT_WINDOW", "deny")

	canUseChannels := SupportsChannelBlock(sdkVersion) && len(channels) > 0

	if canUseChannels {
		// Bloqueo por canal específico
		var successChannels []string
		var lastErr error
		for _, ch := range channels {
			if e := BlockNotifChannel(adbPath, serial, pkg, ch); e != nil {
				lastErr = e
			} else {
				successChannels = append(successChannels, ch)
			}
		}
		if len(successChannels) == 0 {
			if e := DisableNotifications(adbPath, serial, pkg); e != nil {
				return nil, true, fmt.Errorf("channel block failed (%v) and fallback also failed: %w", lastErr, e)
			}
			return nil, true, nil
		}
		return successChannels, false, nil
	}

	// Bloqueo total (SDK < 26 o sin canales identificados)
	if e := DisableNotifications(adbPath, serial, pkg); e != nil {
		return nil, false, fmt.Errorf("failed to disable notifications for %s: %w", pkg, e)
	}
	runADB(adbPath, "-s", serial, "shell", "am", "force-stop", pkg)
	return nil, true, nil
}

// UnblockAdSourceSmart deshace el bloqueo según cómo fue bloqueado:
// - Si blockedChannels no está vacío: desbloquea esos canales específicos
// - Si fullBlocked o sin canales: restaura el package completo
func UnblockAdSourceSmart(adbPath string, serial string, pkg string, blockedChannels []string, sdkVersion string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot unblock system app: %s", pkg)
	}

	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "SYSTEM_ALERT_WINDOW", "allow")

	if len(blockedChannels) > 0 && SupportsChannelBlock(sdkVersion) {
		for _, ch := range blockedChannels {
			UnblockNotifChannel(adbPath, serial, pkg, ch)
		}
		return nil
	}

	return EnableNotifications(adbPath, serial, pkg)
}

// BlockAdSourceFull bloquea TODAS las notificaciones de un package completo,
// independientemente de la versión de Android. Es la opción "nuclear" que el
// usuario puede elegir explícitamente (ej: bloquear Chrome completo).
func BlockAdSourceFull(adbPath string, serial string, pkg string) error {
	if IsSystemProcess("", pkg) {
		return fmt.Errorf("cannot block system app: %s", pkg)
	}
	runADB(adbPath, "-s", serial, "shell", "appops", "set", pkg, "SYSTEM_ALERT_WINDOW", "deny")
	if err := DisableNotifications(adbPath, serial, pkg); err != nil {
		return fmt.Errorf("failed to disable notifications for %s: %w", pkg, err)
	}
	runADB(adbPath, "-s", serial, "shell", "am", "force-stop", pkg)
	return nil
}
