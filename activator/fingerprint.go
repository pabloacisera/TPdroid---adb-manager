package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func getFingerprint() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return windowsFingerprint()
	case "linux":
		return linuxFingerprint()
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func windowsFingerprint() (string, error) {
	cmd := exec.Command("wmic", "baseboard", "get", "serialnumber")
	out, err := cmd.Output()
	if err != nil {
		return fallbackFingerprint()
	}
	raw := strings.TrimSpace(string(out))
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.EqualFold(line, "SerialNumber") {
			return hash(strings.ToUpper(line)), nil
		}
	}
	return fallbackFingerprint()
}

func linuxFingerprint() (string, error) {
	// Try /etc/machine-id (systemd)
	data, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return hash(id), nil
		}
	}

	// Fallback: DMI product_serial
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
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac != "" {
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
