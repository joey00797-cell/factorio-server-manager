package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AppVersion is the build version, set from main ("dev" for local builds).
var AppVersion = "dev"

const latestReleaseAPI = "https://api.github.com/repos/joey00797-cell/factorio-server-manager/releases/latest"

// VersionInfo describes the running build and the newest published release.
type VersionInfo struct {
	Version         string `json:"version"`
	Mode            string `json:"mode"`
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	Notes           string `json:"notes,omitempty"`
	CheckedAt       string `json:"checked_at,omitempty"`
}

var (
	versionMu      sync.Mutex
	versionCache   *VersionInfo
	versionExpires time.Time
)

func installMode() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	return "linux"
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var parts []int
	for _, p := range strings.Split(v, ".") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		parts = append(parts, n)
	}
	return parts
}

func isNewerVersion(latest, current string) bool {
	a, b := parseVersionParts(latest), parseVersionParts(current)
	if a == nil || b == nil {
		return false
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

func fetchLatestRelease() (tag, url, body string, err error) {
	req, err := http.NewRequest("GET", latestReleaseAPI, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fsm-update-check")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", "", err
	}
	return rel.TagName, rel.HTMLURL, rel.Body, nil
}

func currentVersionInfo() VersionInfo {
	cur := AppVersion
	if v := os.Getenv("FSM_VERSION_OVERRIDE"); v != "" {
		cur = v
	}
	info := VersionInfo{Version: cur, Mode: installMode()}
	if cur == "dev" || os.Getenv("FSM_UPDATE_CHECK") == "false" {
		return info
	}

	versionMu.Lock()
	defer versionMu.Unlock()
	if versionCache != nil && time.Now().Before(versionExpires) {
		return *versionCache
	}

	tag, url, body, err := fetchLatestRelease()
	if err != nil {
		log.Printf("update check failed: %v", err)
		versionCache = &info
		versionExpires = time.Now().Add(15 * time.Minute)
		return info
	}
	if r := []rune(body); len(r) > 8000 {
		body = string(r[:8000])
	}
	info.Latest = tag
	info.UpdateAvailable = isNewerVersion(tag, cur)
	info.ReleaseURL = url
	info.Notes = body
	info.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	versionCache = &info
	versionExpires = time.Now().Add(6 * time.Hour)
	return info
}

// GetVersionInfoHandler returns the running version and whether a newer release exists.
func GetVersionInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	WriteResponse(w, currentVersionInfo())
}
