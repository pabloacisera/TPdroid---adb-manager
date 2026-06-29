package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

//go:embed activate.html tpdroid-icon.png tpdroid-icon-without-background.png
var activatorFS embed.FS

var defaultWorkerURL = "http://localhost:8787"

func main() {
	workerURL := os.Getenv("LICENSE_WORKER_URL")
	if workerURL == "" {
		workerURL = defaultWorkerURL
	}

	port := os.Getenv("ACTIVATOR_PORT")
	if port == "" {
		port = "0"
	}

	activated := make(chan struct{}, 1)

	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al iniciar servidor: %v\n", err)
		os.Exit(2)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/activate", func(w http.ResponseWriter, r *http.Request) {
		handleActivate(w, r, workerURL, activated)
	})

	server := &http.Server{Handler: mux}

	// Contexto con timeout de 5 minutos
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Señales
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			server.Close()
		case <-ctx.Done():
			server.Close()
		}
	}()

	// Iniciar servidor HTTP en goroutine
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Error del servidor: %v\n", err)
		}
	}()

	openBrowser(fmt.Sprintf("http://127.0.0.1:%d/", actualPort))
	fmt.Printf("Servidor de activación en http://127.0.0.1:%d/\n", actualPort)
	fmt.Println("Complete la activación en el navegador.")

	// Esperar activación o timeout
	select {
	case <-activated:
		fmt.Println("Activación exitosa.")
		time.Sleep(500 * time.Millisecond)
		server.Close()
		os.Exit(0)
	case <-ctx.Done():
		os.Exit(1)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Serve embedded icon files
	if r.URL.Path == "/tpdroid-icon.png" || r.URL.Path == "/tpdroid-icon-without-background.png" {
		data, err := fs.ReadFile(activatorFS, r.URL.Path[1:])
		if err == nil {
			w.Header().Set("Content-Type", "image/png")
			w.Write(data)
			return
		}
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	html, err := fs.ReadFile(activatorFS, "activate.html")
	if err != nil {
		http.Error(w, "Internal error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

func handleActivate(w http.ResponseWriter, r *http.Request, workerURL string, activated chan<- struct{}) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		Codigo string `json:"codigo"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Solicitud inválida", 400)
		return
	}
	if req.Codigo == "" {
		jsonError(w, "Código de licencia requerido", 400)
		return
	}

	hwID, err := getFingerprint()
	if err != nil {
		jsonError(w, "No se pudo identificar este equipo", 500)
		return
	}

	payload := map[string]string{
		"codigo": req.Codigo,
		"hw_id":  hwID,
	}
	if req.Email != "" {
		payload["email"] = req.Email
	}
	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(workerURL+"/activar", "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		jsonError(w, "No se pudo conectar con el servidor de activación. Verifique su conexión a internet.", 502)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Message string `json:"message"`
		Lic     any    `json:"lic"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Respuesta inválida del servidor", 502)
		return
	}

	if !result.Success {
		jsonError(w, result.Error, 200)
		return
	}

	licPath := licensePath()
	if err := os.MkdirAll(filepath.Dir(licPath), 0700); err != nil {
		jsonError(w, "Error al guardar la licencia", 500)
		return
	}
	licData, _ := json.MarshalIndent(result.Lic, "", "  ")
	if err := os.WriteFile(licPath, licData, 0600); err != nil {
		jsonError(w, "Error al guardar la licencia", 500)
		return
	}

	jsonResponse(w, map[string]any{
		"success": true,
		"message": "Licencia activada para este equipo",
	})

	activated <- struct{}{}
}

// licensePath returns the OS-dependent path for the .lic file.
// The path can be overridden via the TPDROID_LICENSE_PATH environment variable.
// NOTE: This must match backend/licensing/client.go LicensePath() exactly.
// If you change the fallback logic here, update that file too.
func licensePath() string {
	if p := os.Getenv("TPDROID_LICENSE_PATH"); p != "" {
		return p
	}
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "TPDroid", "tpdroid.lic")
	case "linux":
		config := os.Getenv("XDG_CONFIG_HOME")
		if config == "" {
			config = filepath.Join(os.Getenv("HOME"), ".config")
		}
		return filepath.Join(config, "tpdroid", "license.lic")
	default:
		return "tpdroid.lic"
	}
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	}
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   msg,
	})
}
