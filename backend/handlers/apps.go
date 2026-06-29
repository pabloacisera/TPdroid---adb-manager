package handlers

import (
	"net/http"

	"github.com/android-manager/backend/adb"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetApps(c *gin.Context) {
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusOK, []adb.App{})
		return
	}
	apps, err := adb.GetApps(h.AdbPath, serial)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, apps)
}

func (h *Handler) DisableNotification(c *gin.Context) {
	var body struct {
		Package string `json:"package"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot disable notifications for system app"})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}
	if err := adb.DisableNotifications(h.AdbPath, serial, body.Package); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	verified, err := adb.GetNotificationStatus(h.AdbPath, serial, body.Package)
	if err != nil {
		verified = false
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "verified": verified, "message": "Notifications disabled for " + body.Package})
}

func (h *Handler) EnableNotification(c *gin.Context) {
	var body struct {
		Package string `json:"package"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot enable notifications for system app"})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}
	if err := adb.EnableNotifications(h.AdbPath, serial, body.Package); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	verified, err := adb.GetNotificationStatus(h.AdbPath, serial, body.Package)
	if err != nil {
		verified = false
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "verified": !verified, "message": "Notifications enabled for " + body.Package})
}
