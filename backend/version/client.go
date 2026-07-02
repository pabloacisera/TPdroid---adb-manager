package version

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type VersionInfo struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
	NotesES     string `json:"notes_es"`
	NotesEN     string `json:"notes_en"`
	CheckFailed bool   `json:"check_failed"`
}

type Cache struct {
	mu        sync.RWMutex
	workerURL string
	current   string
	cached    VersionInfo
}

func NewCache(workerURL, currentVersion string) *Cache {
	c := &Cache{
		workerURL: workerURL,
		current:   currentVersion,
		cached: VersionInfo{
			Current:     currentVersion,
			Latest:      currentVersion,
			CheckFailed: true,
		},
	}
	go c.startBackgroundChecker()
	c.refresh()
	return c
}

func (c *Cache) Get() VersionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cached
}

func (c *Cache) startBackgroundChecker() {
	interval := 6 * time.Hour
	for {
		time.Sleep(interval)
		c.refresh()
	}
}

func (c *Cache) refresh() {
	if c.workerURL == "" {
		return
	}

	url := strings.TrimRight(c.workerURL, "/") + "/version"
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("VERSION: fetch error: %v", err)
		c.mu.Lock()
		c.cached.CheckFailed = true
		c.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("VERSION: status %d", resp.StatusCode)
		c.mu.Lock()
		c.cached.CheckFailed = true
		c.mu.Unlock()
		return
	}

	var remote struct {
		Latest      string `json:"latest"`
		DownloadURL string `json:"download_url"`
		Changelog   string `json:"changelog"`
		NotesES     string `json:"notes_es"`
		NotesEN     string `json:"notes_en"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&remote); err != nil {
		log.Printf("VERSION: decode error: %v", err)
		return
	}

	if remote.Latest == "" {
		return
	}

	c.mu.Lock()
	c.cached = VersionInfo{
		Current:     c.current,
		Latest:      remote.Latest,
		DownloadURL: remote.DownloadURL,
		Changelog:   remote.Changelog,
		NotesES:     remote.NotesES,
		NotesEN:     remote.NotesEN,
		CheckFailed: false,
	}
	c.mu.Unlock()

	log.Printf("VERSION: latest=%s current=%s", remote.Latest, c.current)
}

// CompareSemver returns true if latest > current using semantic versioning.
func CompareSemver(current, latest string) bool {
	a := parseSemver(current)
	b := parseSemver(latest)
	for i := 0; i < 3; i++ {
		if b[i] > a[i] {
			return true
		}
		if b[i] < a[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err == nil {
			result[i] = n
		}
	}
	return result
}

func HasUpdate(info VersionInfo) bool {
	return CompareSemver(info.Current, info.Latest)
}

func VersionFromTag(tag string) string {
	return strings.TrimPrefix(tag, "v")
}
