package handlers

import (
	"net/http"

	"github.com/android-manager/backend/adb"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetProcesses(c *gin.Context) {
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusOK, []adb.Process{})
		return
	}
	procs, err := adb.GetProcesses(h.AdbPath, serial)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, procs)
}

func (h *Handler) ForceStop(c *gin.Context) {
	var body struct {
		Package string `json:"package"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request"})
		return
	}
	if adb.IsSystemProcess("", body.Package) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Cannot stop system process"})
		return
	}
	serial := h.ResolveSerial()
	if serial == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "no device connected"})
		return
	}
	if err := adb.ForceStop(h.AdbPath, serial, body.Package); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	verified, _ := adb.VerifyProcessStopped(h.AdbPath, serial, body.Package)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"verified": verified,
		"message":  "Process force-stopped",
	})
}
