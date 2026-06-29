package adb

import "testing"

func TestIsSystemProcess_ChromeIsNotSystem(t *testing.T) {
	if IsSystemProcess("", "com.android.chrome") {
		t.Error("com.android.chrome no debe ser considerado proceso de sistema")
	}
}

func TestIsSystemProcess_ChromeWithSystemUID(t *testing.T) {
	if !IsSystemProcess("system", "com.android.chrome") {
		t.Error("com.android.chrome con uid=system SÍ debe ser sistema")
	}
}

func TestIsSystemProcess_SettingsIsSystem(t *testing.T) {
	if !IsSystemProcess("", "com.android.settings") {
		t.Error("com.android.settings debe ser sistema")
	}
}

func TestIsSystemProcess_VendingIsSystem(t *testing.T) {
	if !IsSystemProcess("", "com.android.vending") {
		t.Error("com.android.vending (Play Store) debe estar protegido")
	}
}

func TestIsSystemProcess_YouTubeIsNotSystem(t *testing.T) {
	if IsSystemProcess("", "com.google.android.youtube") {
		t.Error("com.google.android.youtube no debe ser sistema")
	}
}
