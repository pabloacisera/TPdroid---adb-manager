package handlers

import (
	"net/http"

	"github.com/android-manager/backend/version"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetVersion(c *gin.Context) {
	if h.VersionCache == nil {
		c.JSON(http.StatusOK, gin.H{
			"current":         "dev",
			"latest":          "dev",
			"update_available": false,
			"check_failed":     false,
		})
		return
	}
	info := h.VersionCache.Get()
	c.JSON(http.StatusOK, gin.H{
		"current":          info.Current,
		"latest":           info.Latest,
		"download_url":     info.DownloadURL,
		"changelog":        info.Changelog,
		"notes_es":         info.NotesES,
		"notes_en":         info.NotesEN,
		"update_available": version.HasUpdate(info),
		"check_failed":     info.CheckFailed,
	})
}
