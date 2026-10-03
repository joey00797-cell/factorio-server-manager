package factorio

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"context"
	"sync"
	"gorm.io/gorm"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
)

var modsSyncing = struct {
	sync.Mutex
	servers map[string]bool
}{servers: make(map[string]bool)}
var syncCancelMu sync.Mutex
var syncCancelFn context.CancelFunc

func IsModsSyncing() bool {
	return IsModsSyncingForServer(DefaultServerID)
}

func IsModsSyncingForServer(serverID string) bool {
	if serverID == "" {
		serverID = DefaultServerID
	}
	modsSyncing.Lock()
	defer modsSyncing.Unlock()
	return modsSyncing.servers[serverID]
}

func beginModsSync(serverID string) bool {
	if serverID == "" {
		serverID = DefaultServerID
	}
	modsSyncing.Lock()
	defer modsSyncing.Unlock()
	if modsSyncing.servers[serverID] {
		return false
	}
	modsSyncing.servers[serverID] = true
	return true
}

func endModsSync(serverID string) {
	if serverID == "" {
		serverID = DefaultServerID
	}
	modsSyncing.Lock()
	defer modsSyncing.Unlock()
	delete(modsSyncing.servers, serverID)
}

func CancelSync() {
	syncCancelMu.Lock()
	defer syncCancelMu.Unlock()
	if syncCancelFn != nil {
		syncCancelFn()
		syncCancelFn = nil
	}
}

type ModSyncResult struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Status           string `json:"status"` // downloaded, already_installed, builtin, not_found, version_mismatch
	AvailableVersion string `json:"available_version,omitempty"`
}

type ModSyncProgress struct {
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	Current      int             `json:"current,omitempty"`
	Total        int             `json:"total,omitempty"`
	Mod          string          `json:"mod,omitempty"`
	Message      string          `json:"message,omitempty"`
	Warning      string          `json:"warning,omitempty"`
	Mods         []ModSyncResult `json:"mods,omitempty"`
	CurrentBytes int64           `json:"current_bytes,omitempty"`
	TotalBytes   int64           `json:"total_bytes,omitempty"`
}

// baseModNames — моды которые есть в любом vanilla + DLC сейве
var baseModNames = map[string]bool{
	"base":           true,
	"elevated-rails": true,
	"quality":        true,
	"space-age":      true,
	"recycler":       true,
}

// isVanillaSave — true если сейв создан без геймплейных модов
func isVanillaSave(mods []Mod) bool {
	for _, m := range mods {
		if !baseModNames[m.Name] {
			return false
		}
	}
	return true
}

func sendSyncProgress(p ModSyncProgress) {
	sendSyncProgressForServer(DefaultServerID, p)
}

func sendSyncProgressForServer(serverID string, p ModSyncProgress) {
	p.Type = "mods_sync"
	data, _ := json.Marshal(p)
	if serverID == "" || serverID == DefaultServerID {
		websocket.WebsocketHub.GetRoom("mods_sync").Send(string(data))
	}
	if serverID != "" {
		websocket.WebsocketHub.GetRoom("servers:" + serverID + ":mods_sync").Send(string(data))
	}
}

type portalModRelease struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	Version     string `json:"version"`
	FileSize    int64  `json:"file_size"`
}

type portalModInfo struct {
	Releases []portalModRelease `json:"releases"`
}

// normalizeVersion убирает четвёртый компонент если он 0
// version48 читает только 3 части, v[3] всегда 0
// info.json хранит версию как "1.2.3", поэтому приводим к одному формату
func normalizeVersion(v Version) string {
	if v[3] == 0 {
		return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
	}
	return v.String()
}

var ErrModNotOnPortal = fmt.Errorf("mod not available on portal (builtin or DLC)")

// LevelDatMod хранит мод из level.dat0
type LevelDatMod struct {
	Name    string
	Version Version
}

// readModsFromLevelDat читает полный список модов с версиями из level.dat0
// Это работает для всех модов включая добавленные после создания сейва
func readModsFromLevelDat(savePath string) ([]LevelDatMod, error) {
	f, err := OpenArchiveFile(savePath, "level.dat0")
	if err != nil {
		return nil, fmt.Errorf("cannot open level.dat0: %v", err)
	}
	defer f.Close()

	compressed, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read level.dat0: %v", err)
	}

	// Распаковываем zlib
	data, err := zlibDecompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("cannot decompress level.dat0: %v", err)
	}

	// Таблица модов начинается на смещении 0x2c
	if len(data) < 0x2d {
		return nil, fmt.Errorf("level.dat0 too small")
	}

	pos := 0x2c
	n := int(data[pos])
	pos++

	var mods []LevelDatMod
	for i := 0; i < n; i++ {
		if pos >= len(data) {
			break
		}
		nameLen := int(data[pos])
		pos++
		if pos+nameLen+7 > len(data) {
			break
		}
		name := string(data[pos : pos+nameLen])
		pos += nameLen
		v0, v1, v2 := uint(data[pos]), uint(data[pos+1]), uint(data[pos+2])
		pos += 3
		pos += 4 // CRC
		mods = append(mods, LevelDatMod{
			Name:    name,
			Version: Version{v0, v1, v2, 0},
		})
	}

	return mods, nil
}

// zlibDecompress распаковывает zlib данные
func zlibDecompress(data []byte) ([]byte, error) {
	// Пробуем стандартный zlib
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer r.Close()
		return ioutil.ReadAll(r)
	}
	// Пробуем raw deflate
	r2 := flate.NewReader(bytes.NewReader(data))
	defer r2.Close()
	return ioutil.ReadAll(r2)
}

func getModRelease(modName string, version string) (portalModRelease, error) {
	url := fmt.Sprintf("https://mods.factorio.com/api/mods/%s", modName)
	resp, err := http.Get(url)
	if err != nil {
		return portalModRelease{}, fmt.Errorf("portal request failed: %v", err)
	}
	defer resp.Body.Close()

	// 404 — мод не на портале (DLC или встроенный)
	if resp.StatusCode == 404 {
		return portalModRelease{}, ErrModNotOnPortal
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return portalModRelease{}, fmt.Errorf("reading portal response: %v", err)
	}

	// category: internal — встроенный DLC мод, не качаем
	var fullInfo struct {
		Category string             `json:"category"`
		Releases []portalModRelease `json:"releases"`
	}
	if err := json.Unmarshal(body, &fullInfo); err == nil {
		// no-category — DLC заглушка без релизов
		if fullInfo.Category == "no-category" {
			return portalModRelease{}, ErrModNotOnPortal
		}
		// internal без релизов — встроенный DLC (elevated-rails, space-age)
		// internal с релизами — обычный мод (flib)
		if fullInfo.Category == "internal" && len(fullInfo.Releases) == 0 {
			return portalModRelease{}, ErrModNotOnPortal
		}
	}

	var info portalModInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return portalModRelease{}, fmt.Errorf("parsing portal response: %v", err)
	}

	for _, release := range info.Releases {
		if release.Version == version {
			// HEAD запрос для получения размера файла
			var creds Credentials
			if _, err := creds.Load(); err == nil && creds.Username != "" {
				headURL := fmt.Sprintf("https://mods.factorio.com%s?username=%s&token=%s", release.DownloadURL, creds.Username, creds.Userkey)
				if headResp, err := http.Head(headURL); err == nil {
					release.FileSize = headResp.ContentLength
					headResp.Body.Close()
				}
			}
			return release, nil
		}
	}

	return portalModRelease{}, fmt.Errorf("version %s not found for mod %s on portal", version, modName)
}

// SyncModsFromSave читает моды из сейва, сравнивает с установленными,
// качает только недостающие. Прогресс через WebSocket room "mods_sync".
// Пока идёт синк — IsModsSyncing() возвращает true, сервер не стартует.
func SyncModsFromSave(savePath string, modNames []string) {
	modsDir := ""
	if manager := GetServerManager(); manager != nil {
		modsDir = manager.DefaultServer().modsDir()
	}
	if modsDir == "" {
		return
	}
	SyncModsFromSaveForDir(nil, savePath, modsDir, DefaultServerID, modNames, "add")
}

func SyncModsFromSaveForDir(db *gorm.DB, savePath string, modsDir string, serverID string, modNames []string, mode string) {
	if !beginModsSync(serverID) {
		log.Println("SyncModsFromSave: already syncing, skipping")
		sendSyncProgressForServer(serverID, ModSyncProgress{Status: "error", Message: "sync already in progress"})
		return
	}
	defer endModsSync(serverID)

	// 1. Read mods from save
	f, err := OpenArchiveFile(savePath, "level.dat", "level-init.dat")
	if err != nil {
		sendSyncProgressForServer(serverID, ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot open save: %v", err)})
		return
	}
	defer f.Close()

	var header SaveHeader
	if err := header.ReadFrom(f); err != nil {
		sendSyncProgressForServer(serverID, ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot read save header: %v", err)})
		return
	}

	// 1.5. Read full mod list from level.dat0
	levelMods, err := readModsFromLevelDat(savePath)
	if err != nil {
		log.Printf("SyncModsFromSave: cannot read level.dat0, falling back to header: %v", err)
	} else {
		header.Mods = nil
		for _, lm := range levelMods {
			header.Mods = append(header.Mods, Mod{Name: lm.Name, Version: lm.Version})
		}
		log.Printf("SyncModsFromSave: loaded %d mods from level.dat0", len(header.Mods))
	}

	// 1.6. Vanilla save check
	var vanillaWarning string
	if isVanillaSave(header.Mods) {
		vanillaWarning = "Save has no gameplay mods. Mods may have been added later — sync may be incomplete."
		log.Println("SyncModsFromSave: vanilla save detected")
	}

	// Build filter set
	filterMods := make(map[string]bool)
	for _, name := range modNames {
		filterMods[name] = true
	}

	// 2. Process each mod from save
	var results []ModSyncResult
	var toDownload []Mod

	for _, saveMod := range header.Mods {
		if saveMod.Name == "base" {
			continue
		}
		if len(filterMods) > 0 && !filterMods[saveMod.Name] {
			continue
		}
		wantVersion := normalizeVersion(saveMod.Version)

		// DLC/base mods — no file needed
		if baseModNames[saveMod.Name] {
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "already_installed"})
			continue
		}

		if db != nil {
			// Exact version in library?
			var asset ModAsset
			if db.Where("name = ? AND version = ? AND source_type != ?", saveMod.Name, wantVersion, "dlc").First(&asset).Error == nil {
				log.Printf("SyncModsFromSave: %s %s found in library (exact)", saveMod.Name, wantVersion)
				upsertManifestItem(db, serverID, asset.ID, true)
				results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "already_installed"})
				continue
			}
			// Any version in library?
			var anyAsset ModAsset
			if db.Where("name = ? AND source_type != ?", saveMod.Name, "dlc").Order("created_at DESC").First(&anyAsset).Error == nil {
				// Check if portal is available for exact version
				credsCheck := Credentials{}
				portalOk := false
				if ok, err := credsCheck.Load(); err == nil && ok && credsCheck.Username != "" {
					portalOk = true
				}
				if portalOk {
					// Queue for portal download of exact version
					toDownload = append(toDownload, saveMod)
					continue
				}
				log.Printf("SyncModsFromSave: %s version mismatch — save wants %s, library has %s", saveMod.Name, wantVersion, anyAsset.Version)
				results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "version_mismatch", AvailableVersion: anyAsset.Version})
				continue
			}
		}
		// Not in library — need portal
		toDownload = append(toDownload, saveMod)
	}

	// Match save mode: disable mods not in save
	if mode == "match" && db != nil {
		saveModNames := make(map[string]bool)
		for _, r := range results {
			saveModNames[r.Name] = true
		}
		for _, m := range toDownload {
			saveModNames[m.Name] = true
		}
		manifest, mErr := EnsureManifest(db, serverID)
		if mErr == nil {
			var items []ServerModManifestItem
			db.Where("server_mod_manifest_id = ? AND enabled = 1", manifest.ID).Find(&items)
			for _, item := range items {
				var asset ModAsset
				if db.First(&asset, item.ModAssetID).Error == nil {
					if !saveModNames[asset.Name] && asset.SourceType != "dlc" {
						db.Model(&item).Update("enabled", false)
						log.Printf("SyncModsFromSave: disabled %s (not in save)", asset.Name)
					}
				}
			}
		}
	}

	total := len(toDownload)
	log.Printf("SyncModsFromSave: %d mods need portal download", total)

	if total == 0 {
		// Also upsert DLC manifest items
		if db != nil {
			for _, r := range results {
				if r.Status != "already_installed" {
					continue
				}
				if baseModNames[r.Name] {
					var dlcAsset ModAsset
					if db.Where("name = ? AND source_type = ?", r.Name, "dlc").First(&dlcAsset).Error == nil {
						upsertManifestItem(db, serverID, dlcAsset.ID, true)
					}
				}
			}
		}
		sendSyncProgressForServer(serverID, ModSyncProgress{Status: "done", Total: 0, Mods: results, Warning: vanillaWarning})
		return
	}

	// Check portal credentials
	creds := Credentials{}
	portalReady := false
	if ok, err := creds.Load(); err == nil && ok && creds.Username != "" {
		portalReady = true
	}
	log.Printf("SyncModsFromSave: portalReady=%v toDownload=%d", portalReady, total)
	if !portalReady {
		sendSyncProgressForServer(serverID, ModSyncProgress{Status: "error", Message: "Some mods are missing from the library. Log in to the mod portal to download them.", Mods: results})
		return
	}

	// 3. Download missing mods from portal
	sendSyncProgressForServer(serverID, ModSyncProgress{Status: "calculating", Total: total})
	releases := make([]portalModRelease, len(toDownload))
	var totalBytes int64
	for i, saveMod := range toDownload {
		wantVersion := normalizeVersion(saveMod.Version)
		if baseModNames[saveMod.Name] {
			continue
		}
		rel, relErr := getModRelease(saveMod.Name, wantVersion)
		if relErr == nil {
			releases[i] = rel
			totalBytes += rel.FileSize
		}
	}
	var currentBytes int64

	for i, saveMod := range toDownload {
		wantVersion := normalizeVersion(saveMod.Version)
		sendSyncProgressForServer(serverID, ModSyncProgress{
			Status:       "progress",
			Current:      i + 1,
			Total:        total,
			Mod:          saveMod.Name,
			CurrentBytes: currentBytes,
			TotalBytes:   totalBytes,
		})

		if baseModNames[saveMod.Name] {
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "builtin"})
			continue
		}

		release, err := getModRelease(saveMod.Name, wantVersion)
		if err == ErrModNotOnPortal {
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "builtin"})
			continue
		}
		if err != nil {
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "not_found"})
			log.Printf("SyncModsFromSave: %s not found on portal: %v", saveMod.Name, err)
			continue
		}

		currentModIdx := i
		if db != nil {
			asset, libErr := ImportPortalModToLibrary(db, release.DownloadURL, release.FileName, saveMod.Name)
			if libErr != nil {
				results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "not_found"})
				log.Printf("SyncModsFromSave: download failed for %s: %v", saveMod.Name, libErr)
				continue
			}
			upsertManifestItem(db, serverID, asset.ID, true)
		} else {
			mods, modErr := NewMods(modsDir)
			if modErr != nil {
				sendSyncProgressForServer(serverID, ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot refresh mods: %v", modErr)})
				return
			}
			if err = mods.DownloadModWithProgress(release.DownloadURL, release.FileName, saveMod.Name, func(cur int64, tot int64) {
				sendSyncProgressForServer(serverID, ModSyncProgress{
					Status:       "progress",
					Current:      currentModIdx + 1,
					Total:        len(toDownload),
					Mod:          saveMod.Name,
					CurrentBytes: currentBytes + cur,
					TotalBytes:   totalBytes,
				})
			}); err != nil {
				results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "not_found"})
				log.Printf("SyncModsFromSave: download failed for %s: %v", saveMod.Name, err)
				continue
			}
		}

		results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "downloaded"})
		if i < len(releases) {
			currentBytes += releases[i].FileSize
		}
		log.Printf("SyncModsFromSave: downloaded %s %s (%d/%d)", saveMod.Name, wantVersion, i+1, total)
	}

	sendSyncProgressForServer(serverID, ModSyncProgress{Status: "done", Total: total, Mods: results, Warning: vanillaWarning})
}

// upsertManifestItem enables a mod asset in the server manifest
func upsertManifestItem(db *gorm.DB, serverID string, assetID uint, enabled bool) {
	manifest, err := EnsureManifest(db, serverID)
	if err != nil {
		log.Printf("upsertManifestItem: cannot ensure manifest: %v", err)
		return
	}
	// Drop other versions of the same mod from this manifest (library is untouched)
	var targetAsset ModAsset
	if enabled && db.First(&targetAsset, assetID).Error == nil {
		db.Exec(`DELETE FROM server_mod_manifest_items
			WHERE server_mod_manifest_id=?
			AND mod_asset_id != ?
			AND to_delete = 0
			AND mod_asset_id IN (SELECT id FROM mod_assets WHERE name=? AND deleted_at IS NULL)`,
			manifest.ID, assetID, targetAsset.Name)
	}
	var item ServerModManifestItem
	if db.Where("server_mod_manifest_id = ? AND mod_asset_id = ?", manifest.ID, assetID).First(&item).Error == nil {
		db.Model(&item).Update("enabled", enabled)
	} else {
		db.Exec("INSERT INTO server_mod_manifest_items (created_at, updated_at, deleted_at, server_mod_manifest_id, mod_asset_id, enabled, to_delete) VALUES (datetime('now'), datetime('now'), NULL, ?, ?, ?, 0)", manifest.ID, assetID, enabled)
	}
}


// ModStatus describes a mod from save vs installed state
type ModStatus struct {
	Name             string `json:"name"`
	VersionRequired  string `json:"version_required"`
	VersionInstalled string `json:"version_installed"`
	Status           string `json:"status"` // missing, installed, wrong_version, builtin
	PortalURL        string `json:"portal_url"`
}

func GetModsFromSave(savePath string) ([]ModStatus, error) {
	modsDir := ""
	if manager := GetServerManager(); manager != nil {
		modsDir = manager.DefaultServer().modsDir()
	}
	if modsDir == "" {
		return nil, fmt.Errorf("mods directory not configured")
	}
	return GetModsFromSaveForDir(savePath, modsDir)
}

func GetModsFromSaveForDir(savePath string, modsDir string) ([]ModStatus, error) {
	// Читаем моды из level.dat0
	levelMods, err := readModsFromLevelDat(savePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read mods from save: %v", err)
	}

	// Читаем установленные моды
	mods, err := NewMods(modsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read installed mods: %v", err)
	}

	installed := make(map[string]string)
	for _, m := range mods.ModInfoList.Mods {
		installed[m.Name] = m.Version
	}

	var result []ModStatus
	for _, lm := range levelMods {
		if lm.Name == "base" {
			continue
		}

		wantVersion := normalizeVersion(lm.Version)
		status := ModStatus{
			Name:            lm.Name,
			VersionRequired: wantVersion,
			PortalURL:       "https://mods.factorio.com/mod/" + lm.Name,
		}

		if baseModNames[lm.Name] {
			status.Status = "builtin"
		} else if gotVersion, ok := installed[lm.Name]; ok {
			status.VersionInstalled = gotVersion
			if gotVersion == wantVersion {
				status.Status = "installed"
			} else {
				status.Status = "wrong_version"
			}
		} else {
			status.Status = "missing"
		}

		result = append(result, status)
	}

	return result, nil
}
