package handlers

import (
	"sync"
	"time"

	"github.com/android-manager/backend/adb"
	"github.com/android-manager/backend/version"
)

const deviceCacheTTL = 3 * time.Second

type Handler struct {
	mu      sync.RWMutex
	AdbPath string
	Serial  string

	VersionCache *version.Cache

	deviceCache     []adb.DeviceEntry
	deviceCacheTime time.Time
}

func New(adbPath string) *Handler {
	return &Handler{AdbPath: adbPath}
}

func (h *Handler) setSerial(serial string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Serial = serial
}

// GetDevicesCached returns the device list, refreshing from ADB only if the
// cache is stale. This avoids spawning an external process on every request.
func (h *Handler) GetDevicesCached() ([]adb.DeviceEntry, error) {
	h.mu.RLock()
	if h.deviceCache != nil && time.Since(h.deviceCacheTime) < deviceCacheTTL {
		defer h.mu.RUnlock()
		return h.deviceCache, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.deviceCache != nil && time.Since(h.deviceCacheTime) < deviceCacheTTL {
		return h.deviceCache, nil
	}
	devices, err := adb.GetDevices(h.AdbPath)
	if err != nil {
		return nil, err
	}
	h.deviceCache = devices
	h.deviceCacheTime = time.Now()
	return devices, nil
}

func (h *Handler) ResolveSerial() string {
	devices, err := h.GetDevicesCached()
	if err != nil {
		return ""
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Serial != "" {
		for _, d := range devices {
			if d.Serial == h.Serial && d.State == "device" {
				return h.Serial
			}
		}
		h.Serial = ""
	}
	for _, d := range devices {
		if d.State == "device" {
			h.Serial = d.Serial
			return d.Serial
		}
	}
	return ""
}
