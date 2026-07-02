package handlers

import (
	"net/http"

	"github.com/android-manager/backend/adb"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDefinitions(c *gin.Context) {
	defs := adb.GetDefinitions()
	c.JSON(http.StatusOK, defs)
}
