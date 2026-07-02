package adb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefinitions_InitiallyEmpty(t *testing.T) {
	defs := GetDefinitions()
	if defs == nil {
		t.Fatal("GetDefinitions() no debería ser nil")
	}
	if len(defs.KnownNonGamePrefixes) > 0 {
		t.Error("inicialmente KnownNonGamePrefixes debería estar vacío")
	}
}

func TestGetKnownNonGamePrefixes_Fallback(t *testing.T) {
	prefixes := GetKnownNonGamePrefixes()
	if len(prefixes) == 0 {
		t.Error("fallback local de KnownNonGamePrefixes no debería estar vacío")
	}
	if prefixes[0] != "com.whatsapp" {
		t.Errorf("primer prefijo debería ser com.whatsapp, got %s", prefixes[0])
	}
}

func TestGetAdKeywords_Fallback(t *testing.T) {
	keywords := GetAdKeywords()
	if len(keywords) == 0 {
		t.Error("fallback local de AdKeywords no debería estar vacío")
	}
	if keywords[0] != "ad" {
		t.Errorf("primer keyword debería ser ad, got %s", keywords[0])
	}
}

func TestGetGameSegments_Fallback(t *testing.T) {
	segs := GetGameSegments()
	if len(segs) == 0 {
		t.Error("fallback local de GameSegments no debería estar vacío")
	}
}

func TestGetGameEngines_Fallback(t *testing.T) {
	engines := GetGameEngines()
	if len(engines) == 0 {
		t.Error("fallback local de GameEngines no debería estar vacío")
	}
}

func TestFetchAndApplyRemoteDefinitions_EmptyURL(t *testing.T) {
	err := FetchAndApplyRemoteDefinitions("")
	if err != nil {
		t.Errorf("URL vacía no debería dar error, got %v", err)
	}
}

func TestFetchAndApplyRemoteDefinitions_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	err := FetchAndApplyRemoteDefinitions(ts.URL)
	if err == nil {
		t.Error("status 500 debería dar error")
	}
}

func TestFetchAndApplyRemoteDefinitions_EmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	err := FetchAndApplyRemoteDefinitions(ts.URL)
	if err != nil {
		t.Errorf("respuesta vacía no debería dar error, got %v", err)
	}

	// Fallback should still work
	if len(GetKnownNonGamePrefixes()) == 0 {
		t.Error("fallback local debería seguir funcionando tras respuesta vacía")
	}
}

func TestFetchAndApplyRemoteDefinitions_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"known_non_game_prefixes": ["com.test.app"],
			"ad_keywords": ["testad"],
			"game_segments": [".testgame"],
			"game_engines": ["testengine"]
		}`))
	}))
	defer ts.Close()

	err := FetchAndApplyRemoteDefinitions(ts.URL)
	if err != nil {
		t.Fatalf("respuesta válida no debería dar error, got %v", err)
	}

	prefixes := GetKnownNonGamePrefixes()
	if len(prefixes) != 1 || prefixes[0] != "com.test.app" {
		t.Errorf("esperaba [com.test.app], got %v", prefixes)
	}

	keywords := GetAdKeywords()
	if len(keywords) != 1 || keywords[0] != "testad" {
		t.Errorf("esperaba [testad], got %v", keywords)
	}
}

func TestGetSystemUIDs_ReturnsNil(t *testing.T) {
	uids := GetSystemUIDs()
	if uids != nil {
		t.Errorf("sin definiciones remotas, SystemUIDs debería ser nil, got %v", uids)
	}
}

func TestGetSystemPrefixes_ReturnsNil(t *testing.T) {
	prefixes := GetSystemPrefixes()
	if prefixes != nil {
		t.Errorf("sin definiciones remotas, SystemPrefixes debería ser nil, got %v", prefixes)
	}
}
