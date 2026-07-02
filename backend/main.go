package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/android-manager/backend/adb"
	"github.com/android-manager/backend/cors"
	"github.com/android-manager/backend/handlers"
	"github.com/android-manager/backend/licensing"
	"github.com/android-manager/backend/version"
	"github.com/gin-gonic/gin"
)

//go:embed ui
var frontendEmbed embed.FS

var Version = "dev"

func main() {
	adbRel, err := adb.AdbBinaryPath()
	if err != nil {
		log.Fatalf("Failed to resolve ADB binary: %v", err)
	}

	wd, _ := os.Getwd()
	if filepath.Base(wd) == "backend" {
		wd = filepath.Dir(wd)
	}
	adbPath := filepath.Join(wd, adbRel)

	if err := adb.EnsureExecutable(adbPath); err != nil {
		log.Printf("Warning: could not set executable: %v", err)
	}

	h := handlers.New(adbPath)

	// ── Worker URL ─────────────────────────────────────
	workerURL := os.Getenv("LICENSE_WORKER_URL")

	// ── Version cache + definitions fetcher ────────────
	vc := version.NewCache(workerURL, Version)
	h.VersionCache = vc

	go func() {
		if err := adb.FetchAndApplyRemoteDefinitions(workerURL); err != nil {
			log.Printf("DEFINITIONS: %v", err)
		}
	}()

	// ── Validación de licencia ─────────────────────────
	licenseValid := false
	if workerURL == "" {
		licenseValid = true
		log.Println("LICENCIA: modo desarrollo (sin validación)")
	} else if err := licensing.Validate(workerURL); err != nil {
		log.Printf("LICENCIA INVÁLIDA: %v", err)
	} else {
		licenseValid = true
		log.Println("LICENCIA: válida")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors.Middleware())

	if !licenseValid {
		r.Use(licensing.BlockMiddleware())
	} else {
		api := r.Group("/api")
		api.GET("/session-token", func(c *gin.Context) {
			c.JSON(200, gin.H{"token": cors.GetSessionToken()})
		})
		api.Use(cors.SessionTokenMiddleware())
		{
			api.GET("/status", h.GetStatus)
			api.GET("/device", h.GetDevice)
			api.GET("/processes", h.GetProcesses)
			api.POST("/processes/force-stop", h.ForceStop)
			api.GET("/apps", h.GetApps)
			api.POST("/apps/disable-notification", h.DisableNotification)
			api.POST("/apps/enable-notification", h.EnableNotification)
			api.GET("/ads/scan", h.GetAdScan)
			api.POST("/ads/block", h.BlockAdSource)
			api.POST("/ads/unblock", h.UnblockAdSource)
			api.POST("/ads/block-full", h.BlockAdSourceFull)
			api.GET("/version", h.GetVersion)
			api.GET("/definitions", h.GetDefinitions)
		}

		frontendFS, _ := fs.Sub(frontendEmbed, "ui")
		fileServer := http.FileServer(http.FS(frontendFS))

		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.Status(http.StatusNotFound)
				return
			}
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
