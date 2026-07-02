package adb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type RemoteDefinitions struct {
	KnownNonGamePrefixes []string `json:"known_non_game_prefixes"`
	AdKeywords           []string `json:"ad_keywords"`
	GameSegments         []string `json:"game_segments"`
	GameEngines          []string `json:"game_engines"`
	SystemUIDs           []string `json:"system_uids"`
	SystemPrefixes       []string `json:"system_prefixes"`
}

var currentDefinitions atomic.Value

func init() {
	currentDefinitions.Store(&RemoteDefinitions{})
}

func GetDefinitions() *RemoteDefinitions {
	return currentDefinitions.Load().(*RemoteDefinitions)
}

func FetchAndApplyRemoteDefinitions(workerURL string) error {
	if workerURL == "" {
		log.Println("DEFINITIONS: no worker URL, usando fallback local")
		return nil
	}

	url := strings.TrimRight(workerURL, "/") + "/definitions"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("definitions fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("definitions fetch status: %d", resp.StatusCode)
	}

	var defs RemoteDefinitions
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		return fmt.Errorf("definitions decode error: %w", err)
	}

	if len(defs.KnownNonGamePrefixes) == 0 {
		log.Println("DEFINITIONS: respuesta vacía, manteniendo fallback local")
		return nil
	}

	currentDefinitions.Store(&defs)
	log.Printf("DEFINITIONS: actualizadas (%d prefijos)\n", len(defs.KnownNonGamePrefixes))
	return nil
}

func GetKnownNonGamePrefixes() []string {
	defs := GetDefinitions()
	if len(defs.KnownNonGamePrefixes) > 0 {
		return defs.KnownNonGamePrefixes
	}
	return knownNonGamePrefixes
}

func GetAdKeywords() []string {
	defs := GetDefinitions()
	if len(defs.AdKeywords) > 0 {
		return defs.AdKeywords
	}
	return adKeywords
}

func GetGameSegments() []string {
	defs := GetDefinitions()
	if len(defs.GameSegments) > 0 {
		return defs.GameSegments
	}
	return gameSegments
}

func GetGameEngines() []string {
	defs := GetDefinitions()
	if len(defs.GameEngines) > 0 {
		return defs.GameEngines
	}
	return gameEngines
}

func GetSystemUIDs() []string {
	defs := GetDefinitions()
	if len(defs.SystemUIDs) > 0 {
		return defs.SystemUIDs
	}
	return nil
}

func GetSystemPrefixes() []string {
	defs := GetDefinitions()
	if len(defs.SystemPrefixes) > 0 {
		return defs.SystemPrefixes
	}
	return nil
}
