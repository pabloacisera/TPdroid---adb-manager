package handlers

import (
	"net/http"

	"github.com/android-manager/backend/adb"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDevice(c *gin.Context) {
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusOK, gin.H{"authorized": false})
		return
	}
	info, err := adb.GetDeviceInfo(h.AdbPath, serial)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authorized": false})
		return
	}
	c.JSON(http.StatusOK, info)
}
