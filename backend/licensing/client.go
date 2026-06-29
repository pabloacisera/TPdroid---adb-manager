// Package licensing provee validación de licencias contra un Worker Cloudflare.
// Es un módulo autocontenido: copiá esta carpeta a cualquier proyecto Go y funciona.
package licensing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Validate lee el archivo .lic y lo valida contra el Worker.
// Retorna nil si la licencia es válida, o un error describiendo el problema.
func Validate(workerURL string) error {
	licPath := LicensePath()
	licData, err := os.ReadFile(licPath)
	if err != nil {
		return fmt.Errorf("no se encontró archivo de licencia en %s: %w", licPath, err)
	}

	var lic struct {
		Codigo string `json:"codigo"`
		HwID   string `json:"hw_id"`
		Issued string `json:"issued"`
		Hmac   string `json:"hmac"`
	}
	if err := json.Unmarshal(licData, &lic); err != nil {
		return fmt.Errorf("archivo de licencia corrupto: %w", err)
	}

	currentHwID, err := fingerprint()
	if err != nil {
		return fmt.Errorf("no se pudo identificar este equipo: %w", err)
	}

	body, _ := json.Marshal(map[string]any{
		"lic":           lic,
		"current_hw_id": currentHwID,
	})

	resp, err := http.Post(workerURL+"/revalidar", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("no se pudo conectar con el servidor de validación: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("respuesta inválida del servidor: %w", err)
	}

	if !result.Success {
		if result.Error != "" {
			return fmt.Errorf("licencia inválida: %s", result.Error)
		}
		return fmt.Errorf("licencia inválida (código %d)", resp.StatusCode)
	}

	return nil
}

// ─── Fingerprint (duplicado del activator para mantener modularidad) ───

func fingerprint() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return windowsFingerprint()
	case "linux":
		return linuxFingerprint()
	default:
		return "", fmt.Errorf("SO no soportado: %s", runtime.GOOS)
	}
}

func windowsFingerprint() (string, error) {
	cmd := exec.Command("wmic", "baseboard", "get", "serialnumber")
	out, err := cmd.Output()
	if err != nil {
		return fallbackFingerprint()
	}
	raw := strings.TrimSpace(string(out))
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.EqualFold(line, "SerialNumber") {
			return hash(strings.ToUpper(line)), nil
		}
	}
	return fallbackFingerprint()
}

func linuxFingerprint() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return hash(id), nil
		}
	}
	data, err = os.ReadFile("/sys/class/dmi/id/product_serial")
	if err == nil {
		serial := strings.TrimSpace(string(data))
		if serial != "" && !strings.EqualFold(serial, "Not Specified") {
			return hash(strings.ToUpper(serial)), nil
		}
	}
	return fallbackFingerprint()
}

func fallbackFingerprint() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener fingerprint: %w", err)
	}
	var macs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if mac := iface.HardwareAddr.String(); mac != "" {
			macs = append(macs, mac)
		}
	}
	if len(macs) == 0 {
		return "", fmt.Errorf("no se pudo obtener fingerprint")
	}
	return hash(strings.Join(macs, ":")), nil
}

func hash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

// ─── Ruta del archivo .lic ──────────────────────────────

// LicensePath returns the OS-dependent path where the .lic file is stored.
// The path can be overridden via the TPDROID_LICENSE_PATH environment variable.
// NOTE: activator/main.go maintains a copy of this function — if you change
// the fallback logic here, update that file too.
func LicensePath() string {
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
