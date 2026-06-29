package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetStatus(c *gin.Context) {
	devices, err := h.GetDevicesCached()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"connected":     false,
			"device_serial": nil,
		})
		return
	}
	for _, d := range devices {
		if d.State == "device" {
			h.setSerial(d.Serial)
			c.JSON(http.StatusOK, gin.H{
				"connected":     true,
				"device_serial": d.Serial,
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"connected":     false,
		"device_serial": nil,
	})
}
