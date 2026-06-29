package main

import (
	"testing"
)

func TestHashDeterministic(t *testing.T) {
	a := hash("hello")
	b := hash("hello")
	if a != b {
		t.Errorf("hash should be deterministic: %s != %s", a, b)
	}
}

func TestHashLength(t *testing.T) {
	h := hash("test-input")
	if len(h) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %s", len(h), h)
	}
}

func TestHashChangesWithInput(t *testing.T) {
	a := hash("input-a")
	b := hash("input-b")
	if a == b {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashHexOnly(t *testing.T) {
	h := hash("anything")
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex character in hash: %c", c)
		}
	}
}

func TestLicensePathWin(t *testing.T) {
	// Can't easily test without faking runtime.GOOS
	// Just verify the function doesn't panic
	_ = licensePath()
}

func TestGetFingerprintUnsupported(t *testing.T) {
	// getFingerprint uses runtime.GOOS, so on whatever OS we're running
	// it should either return a valid fingerprint or an error
	fp, err := getFingerprint()
	if err != nil {
		t.Logf("Fingerprint error (acceptable in CI): %v", err)
	} else {
		if len(fp) != 64 {
			t.Errorf("expected 64 hex chars, got %d: %s", len(fp), fp)
		}
	}
}
