package version

import "testing"

func TestCompareSemver_Newer(t *testing.T) {
	if !CompareSemver("0.1.0", "0.2.0") {
		t.Error("0.2.0 debería ser más nueva que 0.1.0")
	}
	if !CompareSemver("1.0.0", "2.0.0") {
		t.Error("2.0.0 debería ser más nueva que 1.0.0")
	}
	if !CompareSemver("0.1.9", "0.2.0") {
		t.Error("0.2.0 debería ser más nueva que 0.1.9")
	}
}

func TestCompareSemver_Older(t *testing.T) {
	if CompareSemver("0.2.0", "0.1.0") {
		t.Error("0.1.0 no debería ser más nueva que 0.2.0")
	}
	if CompareSemver("2.0.0", "1.0.0") {
		t.Error("1.0.0 no debería ser más nueva que 2.0.0")
	}
}

func TestCompareSemver_Equal(t *testing.T) {
	if CompareSemver("0.2.0", "0.2.0") {
		t.Error("versiones iguales no deberían reportar update")
	}
}

func TestCompareSemver_WithVPrefix(t *testing.T) {
	if !CompareSemver("v0.1.0", "v0.2.0") {
		t.Error("v0.2.0 debería ser más nueva que v0.1.0 con prefijo v")
	}
	if CompareSemver("v0.2.0", "v0.1.0") {
		t.Error("v0.1.0 no debería ser más nueva que v0.2.0 con prefijo v")
	}
}

func TestCompareSemver_MixedPrefix(t *testing.T) {
	if !CompareSemver("0.1.0", "v0.2.0") {
		t.Error("v0.2.0 debería ser más nueva que 0.1.0 con prefijos mixtos")
	}
	if CompareSemver("v0.2.0", "0.1.0") {
		t.Error("0.1.0 no debería ser más nueva que v0.2.0 con prefijos mixtos")
	}
}

func TestCompareSemver_Malformed(t *testing.T) {
	// current malformed → parsed as [0,0,0], so any real version is an update
	if !CompareSemver("blah", "0.2.0") {
		t.Error("current malformado [0,0,0] debería considerar 0.2.0 como update")
	}
	// latest malformed → parsed as [0,0,0], cannot be newer than [0,1,0]
	if CompareSemver("0.1.0", "notaversion") {
		t.Error("latest malformado [0,0,0] no debería ser update de 0.1.0")
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"0.1.0", [3]int{0, 1, 0}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"2.0.0", [3]int{2, 0, 0}},
		{"10.20.30", [3]int{10, 20, 30}},
		{"blah", [3]int{0, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}
	for _, tc := range tests {
		got := parseSemver(tc.input)
		if got != tc.want {
			t.Errorf("parseSemver(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestHasUpdate(t *testing.T) {
	if HasUpdate(VersionInfo{Current: "0.1.0", Latest: "0.1.0"}) {
		t.Error("misma versión no debería tener update")
	}
	if !HasUpdate(VersionInfo{Current: "0.1.0", Latest: "0.2.0"}) {
		t.Error("0.2.0 debería ser update de 0.1.0")
	}
	if HasUpdate(VersionInfo{Current: "0.2.0", Latest: "0.1.0"}) {
		t.Error("0.1.0 no debería ser update de 0.2.0")
	}
}

func TestVersionFromTag(t *testing.T) {
	if got := VersionFromTag("v0.2.0"); got != "0.2.0" {
		t.Errorf("VersionFromTag(v0.2.0) = %q, want 0.2.0", got)
	}
	if got := VersionFromTag("0.2.0"); got != "0.2.0" {
		t.Errorf("VersionFromTag(0.2.0) = %q, want 0.2.0", got)
	}
}
