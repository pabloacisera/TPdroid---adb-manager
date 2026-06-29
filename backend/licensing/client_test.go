package licensing

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
	h := hash("test")
	if len(h) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(h))
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

func TestHashChanges(t *testing.T) {
	if hash("a") == hash("b") {
		t.Error("different inputs should produce different hashes")
	}
}

func TestLicensePathReturnsString(t *testing.T) {
	path := LicensePath()
	if path == "" {
		t.Error("LicensePath() should not return empty string")
	}
	t.Logf("License path: %s", path)
}

func TestValidateNoLicFile(t *testing.T) {
	err := Validate("http://localhost:8787")
	if err == nil {
		t.Error("Validate() should fail when no .lic file exists")
	}
	t.Logf("Expected error: %v", err)
}
