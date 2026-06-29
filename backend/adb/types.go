package adb

type DeviceEntry struct {
	Serial string
	State  string
}

type DeviceInfo struct {
	Serial         string `json:"serial"`
	Brand          string `json:"brand"`
	Model          string `json:"model"`
	AndroidVersion string `json:"android_version"`
	SDKVersion     string `json:"sdk_version"`
	Authorized     bool   `json:"authorized"`
}

type Process struct {
	PID           string `json:"pid"`
	Name          string `json:"name"`
	UID           string `json:"uid"`
	Status        string `json:"status"`
	SystemProcess bool   `json:"system_process"`
}

type App struct {
	Package               string   `json:"package"`
	Label                 string   `json:"label"`
	SystemApp             bool     `json:"system_app"`
	IsGame                bool     `json:"is_game"`
	Permissions           []string `json:"permissions"`
	NotificationsDisabled bool     `json:"notifications_disabled"`
}

// AdEntry representa una fuente de publicidad o notificación intrusiva detectada dinámicamente.
type AdEntry struct {
	Package          string   `json:"package"`
	Reasons          []string `json:"reasons"`
	NotifChannels    []string `json:"notif_channels"`
	AlarmTags        []string `json:"alarm_tags"`
	NotifBlocked     bool     `json:"notif_blocked"`
	OverlayRevoked   bool     `json:"overlay_revoked"`
	IsSystemApp      bool     `json:"is_system_app"`
	BlockedChannels  []string `json:"blocked_channels"`
	FullBlocked      bool     `json:"full_blocked"`
}
