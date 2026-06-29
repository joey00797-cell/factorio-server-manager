package factorio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"bytes"
	"compress/flate"
	"compress/zlib"
	"net/http"
	"context"
	"sync"
	"sync/atomic"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

var modsSyncing atomic.Bool
var syncCancelMu sync.Mutex
var syncCancelFn context.CancelFunc

func IsModsSyncing() bool {
	return modsSyncing.Load()
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
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"` // downloaded, already_installed, builtin, not_found
}

type ModSyncProgress struct {
	Type         string        `json:"type"`
	Status       string        `json:"status"`
	Current      int           `json:"current,omitempty"`
	Total        int           `json:"total,omitempty"`
	Mod          string        `json:"mod,omitempty"`
	Message      string        `json:"message,omitempty"`
	Warning      string        `json:"warning,omitempty"`
	Mods         []ModSyncResult `json:"mods,omitempty"`
	CurrentBytes int64         `json:"current_bytes,omitempty"`
	TotalBytes   int64         `json:"total_bytes,omitempty"`
}

// baseModNames — моды которые есть в любом vanilla + DLC сейве
var baseModNames = map[string]bool{
	"base":           true,
	"elevated-rails": true,
	"quality":        true,
	"space-age":      true,
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
	p.Type = "mods_sync"
	data, _ := json.Marshal(p)
	room := websocket.WebsocketHub.GetRoom("mods_sync")
	room.Send(string(data))
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
	n := int(data[pos]); pos++

	var mods []LevelDatMod
	for i := 0; i < n; i++ {
		if pos >= len(data) {
			break
		}
		nameLen := int(data[pos]); pos++
		if pos+nameLen+7 > len(data) {
			break
		}
		name := string(data[pos : pos+nameLen]); pos += nameLen
		v0, v1, v2 := uint(data[pos]), uint(data[pos+1]), uint(data[pos+2]); pos += 3
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
	if !modsSyncing.CompareAndSwap(false, true) {
		log.Println("SyncModsFromSave: already syncing, skipping")
		sendSyncProgress(ModSyncProgress{Status: "error", Message: "sync already in progress"})
		return
	}
	defer modsSyncing.Store(false)

	config := bootstrap.GetConfig()

	// 1. Читаем список модов из сейва
	f, err := OpenArchiveFile(savePath, "level.dat", "level-init.dat")
	if err != nil {
		sendSyncProgress(ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot open save: %v", err)})
		return
	}
	defer f.Close()

	var header SaveHeader
	if err := header.ReadFrom(f); err != nil {
		sendSyncProgress(ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot read save header: %v", err)})
		return
	}

	// 1.5. Читаем полный список модов из level.dat0
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

		// 1.6. Проверяем тип сейва
	var vanillaWarning string
	if isVanillaSave(header.Mods) {
		vanillaWarning = "Сейв создан без геймплейных модов. Моды могли быть добавлены позже — синхронизация может быть неполной."
		log.Println("SyncModsFromSave: vanilla save detected, mods may have been added later")
	}

	// 2. Читаем установленные моды
	mods, err := NewMods(config.FactorioModsDir)
	if err != nil {
		sendSyncProgress(ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot read installed mods: %v", err)})
		return
	}

	// 3. Map установленных: name -> version (из info.json, формат "1.2.3")
	installed := make(map[string]string)
	for _, m := range mods.ModInfoList.Mods {
		installed[m.Name] = m.Version
	}

	// Строим set модов для скачивания если передан список
	filterMods := make(map[string]bool)
	for _, name := range modNames {
		filterMods[name] = true
	}

	// 4. Обходим все моды из сейва — собираем результаты
	var results []ModSyncResult
	var toDownload []Mod

	for _, saveMod := range header.Mods {
		if saveMod.Name == "base" {
			continue
		}
		// Если передан список — качаем только выбранные
		if len(filterMods) > 0 && !filterMods[saveMod.Name] {
			continue
		}
		wantVersion := normalizeVersion(saveMod.Version)
		if gotVersion, ok := installed[saveMod.Name]; ok && gotVersion == wantVersion {
			log.Printf("SyncModsFromSave: %s %s already installed, skipping", saveMod.Name, wantVersion)
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "already_installed"})
			continue
		}
		toDownload = append(toDownload, saveMod)
	}

	total := len(toDownload)
	log.Printf("SyncModsFromSave: %d mods to download", total)

	if total == 0 {
		sendSyncProgress(ModSyncProgress{Status: "done", Total: 0, Mods: results, Warning: vanillaWarning})
		return
	}

	// 5. Собираем релизы и считаем общий размер
	sendSyncProgress(ModSyncProgress{Status: "calculating", Total: total})
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

// 6. Качаем недостающие
	for i, saveMod := range toDownload {
		wantVersion := normalizeVersion(saveMod.Version)

		sendSyncProgress(ModSyncProgress{
			Status:       "progress",
			Current:      i + 1,
			Total:        total,
			Mod:          saveMod.Name,
			CurrentBytes: currentBytes,
			TotalBytes:   totalBytes,
		})

		// Сначала проверяем локальный список DLC/базовых модов
		if baseModNames[saveMod.Name] {
			log.Printf("SyncModsFromSave: %s is builtin/DLC (local list), skipping", saveMod.Name)
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "builtin"})
			continue
		}

		release, err := getModRelease(saveMod.Name, wantVersion)
		if err == ErrModNotOnPortal {
			log.Printf("SyncModsFromSave: %s is builtin/DLC, skipping", saveMod.Name)
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "builtin"})
			continue
		}
		if err != nil {
			results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "not_found"})
			log.Printf("SyncModsFromSave: %s not found on portal: %v", saveMod.Name, err)
			continue
		}

		mods, err = NewMods(config.FactorioModsDir)
		if err != nil {
			sendSyncProgress(ModSyncProgress{Status: "error", Message: fmt.Sprintf("cannot refresh mods: %v", err)})
			return
		}

		currentModIdx := i
		if err = mods.DownloadModWithProgress(release.DownloadURL, release.FileName, saveMod.Name, func(cur int64, total int64) {
			sendSyncProgress(ModSyncProgress{
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

		results = append(results, ModSyncResult{Name: saveMod.Name, Version: wantVersion, Status: "downloaded"})
		if i < len(releases) {
			currentBytes += releases[i].FileSize
		}
		log.Printf("SyncModsFromSave: downloaded %s %s (%d/%d)", saveMod.Name, wantVersion, i+1, total)
	}

	// Обновляем mod-list.json — включаем нужные, выключаем лишние
	finalMods, err := NewMods(config.FactorioModsDir)
	if err == nil {
		// Строим set модов из сейва
		saveModSet := make(map[string]bool)
		for _, saveMod := range header.Mods {
			if saveMod.Name != "base" {
				saveModSet[saveMod.Name] = true
			}
		}
		// Включаем моды из сейва, выключаем остальные
		for i, m := range finalMods.ModSimpleList.Mods {
			if m.Name == "base" {
				continue
			}
			if saveModSet[m.Name] {
				finalMods.ModSimpleList.Mods[i].Enabled = true
			} else {
				finalMods.ModSimpleList.Mods[i].Enabled = false
			}
		}
		if saveErr := finalMods.ModSimpleList.saveModInfoJson(); saveErr != nil {
			log.Printf("SyncModsFromSave: error saving mod-list.json: %v", saveErr)
		}
	}

	sendSyncProgress(ModSyncProgress{Status: "done", Total: total, Mods: results, Warning: vanillaWarning})
}

// ModStatus — статус мода при сравнении сейва с установленными
type ModStatus struct {
	Name            string `json:"name"`
	VersionRequired string `json:"version_required"` // версия в сейве
	VersionInstalled string `json:"version_installed"` // версия установленная (если есть)
	Status          string `json:"status"` // missing, installed, wrong_version, builtin
	PortalURL       string `json:"portal_url"`
}

// GetModsFromSave читает моды из сейва и сравнивает с установленными
func GetModsFromSave(savePath string) ([]ModStatus, error) {
	config := bootstrap.GetConfig()

	// Читаем моды из level.dat0
	levelMods, err := readModsFromLevelDat(savePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read mods from save: %v", err)
	}

	// Читаем установленные моды
	mods, err := NewMods(config.FactorioModsDir)
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
