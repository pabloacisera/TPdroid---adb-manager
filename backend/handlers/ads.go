package handlers

import (
	"net/http"

	"github.com/android-manager/backend/adb"
	"github.com/gin-gonic/gin"
)

// GetAdScan — GET /api/ads/scan
func (h *Handler) GetAdScan(c *gin.Context) {
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusOK, []adb.AdEntry{})
		return
	}
	entries, err := adb.ScanAdSources(h.AdbPath, serial)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

// BlockAdSource — POST /api/ads/block
// ACTUALIZADO: bloqueo inteligente por canal si el dispositivo lo soporta.
// Body: { "package": "...", "channels": ["channelId1", ...], "sdk_version": "33" }
func (h *Handler) BlockAdSource(c *gin.Context) {
	var body struct {
		Package    string   `json:"package"`
		Channels   []string `json:"channels"`
		SDKVersion string   `json:"sdk_version"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Package == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: package required"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot block system app: " + body.Package})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}

	blockedChannels, fullBlocked, err := adb.BlockAdSourceSmart(h.AdbPath, serial, body.Package, body.Channels, body.SDKVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "Blocked: " + body.Package,
		"blocked_channels": blockedChannels,
		"full_blocked":     fullBlocked,
	})
}

// UnblockAdSource — POST /api/ads/unblock
// ACTUALIZADO: desbloqueo simétrico (por canal o total según cómo fue bloqueado).
// Body: { "package": "...", "blocked_channels": ["channelId1", ...], "sdk_version": "33" }
func (h *Handler) UnblockAdSource(c *gin.Context) {
	var body struct {
		Package         string   `json:"package"`
		BlockedChannels []string `json:"blocked_channels"`
		SDKVersion      string   `json:"sdk_version"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Package == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: package required"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot unblock system app: " + body.Package})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}

	if err := adb.UnblockAdSourceSmart(h.AdbPath, serial, body.Package, body.BlockedChannels, body.SDKVersion); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Unblocked: " + body.Package})
}

// BlockAdSourceFull — POST /api/ads/block-full
// Bloqueo TOTAL del package: todas sus notificaciones, independientemente de canales o SDK.
// El usuario lo elige explícitamente. Para browsers: bloquea TODAS las notificaciones del navegador.
// Body: { "package": "..." }
func (h *Handler) BlockAdSourceFull(c *gin.Context) {
	var body struct {
		Package string `json:"package"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Package == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request: package required"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot block system app: " + body.Package})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}

	if err := adb.BlockAdSourceFull(h.AdbPath, serial, body.Package); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Full block applied: " + body.Package,
		"full_blocked": true,
	})
}
