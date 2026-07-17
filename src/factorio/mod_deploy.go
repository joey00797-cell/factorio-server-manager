package factorio

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ensureManifest возвращает манифест сервера, создаёт если нет
func ensureManifest(db *gorm.DB, serverID string) (ServerModManifest, error) {
	var manifest ServerModManifest
	err := db.Preload("Items.ModAsset").Where("server_id = ?", serverID).First(&manifest).Error
	if err == gorm.ErrRecordNotFound {
		manifest = ServerModManifest{
			ServerID:       serverID,
			SettingsSource: "preserve",
		}
		if createErr := db.Create(&manifest).Error; createErr != nil {
			// Race condition: другой запрос уже создал манифест
			if err2 := db.Where("server_id = ?", serverID).First(&manifest).Error; err2 != nil {
				return manifest, createErr
			}
			return manifest, nil
		}
		return manifest, nil
	}
	return manifest, err
}

// GetManifest возвращает манифест сервера для API
func GetManifest(db *gorm.DB, serverID string) (ManifestResult, error) {
	manifest, err := ensureManifest(db, serverID)
	if err != nil {
		return ManifestResult{}, err
	}
	return newManifestResult(manifest), nil
}

func newManifestResult(manifest ServerModManifest) ManifestResult {
	items := make([]ManifestItemResult, len(manifest.Items))
	for i, item := range manifest.Items {
		items[i] = newManifestItemResult(item)
	}
	return ManifestResult{
		ServerID:       manifest.ServerID,
		SettingsSource: manifest.SettingsSource,
		LastAppliedAt:  manifest.LastAppliedAt,
		Items:          items,
	}
}

// UpdateManifestItems обновляет список модов в манифесте
func UpdateManifestItems(db *gorm.DB, serverID string, items []struct {
	AssetID   uint `json:"asset_id"`
	Enabled   bool `json:"enabled"`
	ToDelete  bool `json:"to_delete"`
}) (ManifestResult, error) {
	manifest, err := ensureManifest(db, serverID)
	if err != nil {
		return ManifestResult{}, err
	}

	// Удаляем старые items физически включая soft-deleted
	if err := db.Exec("DELETE FROM server_mod_manifest_items WHERE server_mod_manifest_id = ?", manifest.ID).Error; err != nil {
		return ManifestResult{}, err
	}
	// Также чистим orphaned soft-deleted записи для этого манифеста
	db.Exec("DELETE FROM server_mod_manifest_items WHERE server_mod_manifest_id = ? AND deleted_at IS NOT NULL", manifest.ID)

	// Создаём новые
	for _, item := range items {
		if err := db.Exec(
			"INSERT INTO server_mod_manifest_items (created_at, updated_at, deleted_at, server_mod_manifest_id, mod_asset_id, enabled, to_delete) VALUES (datetime('now'), datetime('now'), NULL, ?, ?, ?, ?)",
			manifest.ID, item.AssetID, item.Enabled, item.ToDelete,
		).Error; err != nil {
			return ManifestResult{}, err
		}
	}

	return GetManifest(db, serverID)
}

// deployedState читает что реально лежит в папке mods сервера
// ResetManifestToDeployed resets manifest to match what is actually on server disk
func ResetManifestToDeployed(db *gorm.DB, serverID string) (ManifestResult, error) {
	serverObj, ok := GetServerManager().GetServer(serverID)
	if !ok {
		return ManifestResult{}, fmt.Errorf("server not found: %s", serverID)
	}

	deployed, err := deployedState(serverObj)
	if err != nil {
		return ManifestResult{}, err
	}

	manifest, err := ensureManifest(db, serverID)
	if err != nil {
		return ManifestResult{}, err
	}

	// Clear current manifest items
	db.Unscoped().Where("server_mod_manifest_id = ?", manifest.ID).Delete(&ServerModManifestItem{})

	// Rebuild from deployed state
	for _, d := range deployed {
		if d.Name == "base" {
			continue
		}
		var asset ModAsset
		if err := db.Where("name = ? AND version = ?", d.Name, d.Version).First(&asset).Error; err != nil {
			continue
		}
		db.Create(&ServerModManifestItem{
			ServerModManifestID: manifest.ID,
			ModAssetID:          asset.ID,
			Enabled:             true,
		})
	}

	return GetManifest(db, serverID)
}

func deployedState(server *Server) ([]ModDeployState, error) {
	modsDir := server.Paths.ModsDir
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Читаем mod-list.json для статуса enabled
	enabledMap := map[string]bool{}
	modListPath := filepath.Join(modsDir, "mod-list.json")
	if data, err := os.ReadFile(modListPath); err == nil {
		var modList struct {
			Mods []struct {
				Name    string `json:"name"`
				Enabled bool   `json:"enabled"`
			} `json:"mods"`
		}
		if err := jsonUnmarshal(data, &modList); err == nil {
			for _, m := range modList.Mods {
				enabledMap[m.Name] = m.Enabled
			}
		}
	}

	var states []ModDeployState

	// Add DLC mods from mod-list.json (they have no zip file)
	dlcNames := map[string]bool{"space-age": true, "elevated-rails": true, "quality": true}
	for name, enabled := range enabledMap {
		if dlcNames[name] {
			states = append(states, ModDeployState{Name: name, Version: "", Enabled: enabled, IsDLC: true})
		}
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".zip" {
			continue
		}
		// Parse filename: modname_version.zip
		base := strings.TrimSuffix(entry.Name(), ".zip")
		lidx := strings.LastIndex(base, "_")
		if lidx < 0 {
			continue
		}
		name := base[:lidx]
		version := base[lidx+1:]
		enabled, ok := enabledMap[name]
		if !ok {
			enabled = true
		}
		states = append(states, ModDeployState{Name: name, Version: version, Enabled: enabled})
	}
	return states, nil
}

// desiredDeployState вычисляет желаемое состояние из манифеста + валидирует зависимости
func desiredDeployState(db *gorm.DB, manifest ServerModManifest, baseVersion string) ([]ModDeployState, []ModPreviewIssue, error) {
	var items []ServerModManifestItem
	if err := db.Preload("ModAsset").Where("server_mod_manifest_id = ?", manifest.ID).Find(&items).Error; err != nil {
		return nil, nil, err
	}

	enabledAssetsByName := map[string]ModAsset{}
	for _, item := range items {
		if item.Enabled {
			enabledAssetsByName[item.ModAsset.Name] = item.ModAsset
		}
	}

	var issues []ModPreviewIssue
	var states []ModDeployState
	for _, item := range items {
		// ToDelete items are excluded from desired state -> appear in ToRemove
		if item.ToDelete {
			continue
		}
		assetIssues := validateAssetDependencies(item.ModAsset, enabledAssetsByName, baseVersion)
		issues = append(issues, assetIssues...)
		id := item.ModAssetID
		states = append(states, ModDeployState{
			Name:    item.ModAsset.Name,
			Version: item.ModAsset.Version,
			Enabled: item.Enabled,
			AssetID: &id,
			IsDLC:   item.ModAsset.SourceType == "dlc",
		})
	}
	return states, issues, nil
}

// validateAssetDependencies проверяет зависимости мода
func validateAssetDependencies(asset ModAsset, enabledByName map[string]ModAsset, baseVersion string) []ModPreviewIssue {
	var issues []ModPreviewIssue

	for _, dep := range asset.Dependencies() {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}

		optional := false
		incompatible := false
		switch {
		case strings.HasPrefix(dep, "!"):
			incompatible = true
			dep = trimDepPrefix(dep, "!")
		case strings.HasPrefix(dep, "(?)"):
			optional = true
			dep = trimDepPrefix(dep, "(?)")
		case strings.HasPrefix(dep, "?"), strings.HasPrefix(dep, "~"):
			optional = true
			dep = dep[1:]
			dep = strings.TrimSpace(dep)
		}

		parts := strings.Fields(dep)
		if len(parts) == 0 {
			continue
		}
		name := parts[0]

		if incompatible {
			if _, exists := enabledByName[name]; exists {
				issues = append(issues, ModPreviewIssue{
					Level:   "error",
					Code:    "incompatible_mod",
					Message: fmt.Sprintf("Mod %s %s incompatible with %s", asset.Name, asset.Version, name),
				})
			}
			continue
		}

		if name == "base" {
			continue // версию base проверяем отдельно если нужно
		}

		if _, exists := enabledByName[name]; !exists && !optional {
			issues = append(issues, ModPreviewIssue{
				Level:   "error",
				Code:    "missing_dependency",
				Message: fmt.Sprintf("Mod %s %s requires %s", asset.Name, asset.Version, name),
			})
		}
	}
	return issues
}

// PreviewApply вычисляет diff между желаемым и реальным состоянием
func PreviewApply(db *gorm.DB, serverID string) (ModApplyPreview, error) {
	manifest, err := ensureManifest(db, serverID)
	if err != nil {
		return ModApplyPreview{}, err
	}

	serverObj, ok := GetServerManager().GetServer(serverID)
	if !ok {
		return ModApplyPreview{}, fmt.Errorf("server not found: %s", serverID)
	}
	server := serverObj

	desired, issues, err := desiredDeployState(db, manifest, server.BaseModVersion)
	if err != nil {
		return ModApplyPreview{}, err
	}

	deployed, err := deployedState(server)
	if err != nil {
		return ModApplyPreview{}, err
	}

	preview := ModApplyPreview{
		ServerID:        serverID,
		CanApply:        len(issues) == 0 && !server.GetRunning(),
		NeedsServerStop: server.GetRunning(),
		Desired:         desired,
		Deployed:        deployed,
		Issues:          issues,
	}

	if server.GetRunning() {
		preview.Issues = append(preview.Issues, ModPreviewIssue{
			Level:   "error",
			Code:    "server_running",
			Message: "Server must be stopped before applying mods.",
		})
	}

	// Вычисляем diff
	desiredByKey := map[string]ModDeployState{}
	deployedByKey := map[string]ModDeployState{}
	for _, s := range desired {
		desiredByKey[s.Name+"@"+s.Version] = s
	}
	for _, s := range deployed {
		deployedByKey[s.Name+"@"+s.Version] = s
	}

	for key, d := range desiredByKey {
		// DLC моды не имеют ZIP файла, сравниваем только enabled статус
		if d.IsDLC {
			depl, ok := deployedByKey[d.Name+"@"]
			if ok {
				if d.Enabled && !depl.Enabled {
					preview.ToEnable = append(preview.ToEnable, d)
				} else if !d.Enabled && depl.Enabled {
					preview.ToDisable = append(preview.ToDisable, d)
				}
			}
			continue
		}
		depl, ok := deployedByKey[key]
		if !ok {
			preview.ToAdd = append(preview.ToAdd, d)
			continue
		}
		if d.Enabled && !depl.Enabled {
			preview.ToEnable = append(preview.ToEnable, d)
		} else if !d.Enabled && depl.Enabled {
			preview.ToDisable = append(preview.ToDisable, d)
		}
	}
	for key, d := range deployedByKey {
		if _, ok := desiredByKey[key]; !ok {
			preview.ToRemove = append(preview.ToRemove, d)
		}
	}

	// Include to_delete manifest items that are not deployed on disk
	// (e.g. added to library but never applied) so they still appear in ToRemove
	// and get cleaned up from the library on Apply
	var toDeleteItems []ServerModManifestItem
	if err := db.Preload("ModAsset").Where("server_mod_manifest_id = ? AND to_delete = ?", manifest.ID, true).Find(&toDeleteItems).Error; err == nil {
		for _, item := range toDeleteItems {
			key := item.ModAsset.Name + "@" + item.ModAsset.Version
			if _, inDeployed := deployedByKey[key]; !inDeployed {
				id := item.ModAssetID
				preview.ToRemove = append(preview.ToRemove, ModDeployState{
					Name:    item.ModAsset.Name,
					Version: item.ModAsset.Version,
					Enabled: false,
					AssetID: &id,
				})
			}
		}
	}

	sortModStates(preview.ToAdd)
	sortModStates(preview.ToRemove)
	sortModStates(preview.ToEnable)
	sortModStates(preview.ToDisable)

	return preview, nil
}

// ApplyManifest применяет манифест — деплоит моды в папку сервера
func ApplyManifest(db *gorm.DB, serverID string) (ModApplyPreview, error) {
	preview, err := PreviewApply(db, serverID)
	if err != nil {
		return ModApplyPreview{}, err
	}
	if !preview.CanApply {
		return preview, nil
	}

	serverObj, ok := GetServerManager().GetServer(serverID)
	if !ok {
		return ModApplyPreview{}, fmt.Errorf("server not found: %s", serverID)
	}
	server := serverObj

	manifest, err := ensureManifest(db, serverID)
	if err != nil {
		return ModApplyPreview{}, err
	}

	modsDir := server.Paths.ModsDir
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return ModApplyPreview{}, err
	}

	// Чистим папку mods — удаляем все zip и mod-list.json
	entries, _ := os.ReadDir(modsDir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".zip" || name == "mod-list.json" {
			os.Remove(filepath.Join(modsDir, name))
		}
	}

	// Копируем нужные моды из библиотеки
	modList := ModSimpleList{
		Destination: modsDir,
		Mods: []ModSimple{{Name: "base", Enabled: true}},
	}

	var items []ServerModManifestItem
	db.Preload("ModAsset").Where("server_mod_manifest_id = ?", manifest.ID).Find(&items)

	for _, item := range items {
		// Skip mods marked for deletion
		if item.ToDelete {
			continue
		}
		// DLC моды не копируем, только добавляем в mod-list.json
		if item.ModAsset.SourceType == "dlc" {
			modList.Mods = append(modList.Mods, ModSimple{
				Name:    item.ModAsset.Name,
				Enabled: item.Enabled,
			})
			continue
		}
		// Skip if ModAsset not loaded (deleted from library)
		if item.ModAsset.ID == 0 {
			log.Printf("Warning: mod asset id=%d not found in library, skipping", item.ModAssetID)
			continue
		}
		src, err := os.Open(item.ModAsset.ArtifactPath)
		if err != nil {
			log.Printf("Warning: cannot open mod asset %s: %v", item.ModAsset.ArtifactPath, err)
			continue
		}
		dstPath := filepath.Join(modsDir, item.ModAsset.FileName)
		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			return ModApplyPreview{}, err
		}
		io.Copy(dst, src)
		src.Close()
		dst.Close()

		modList.Mods = append(modList.Mods, ModSimple{
			Name:    item.ModAsset.Name,
			Enabled: item.Enabled,
		})
	}

	if err := modList.saveModInfoJson(); err != nil {
		return ModApplyPreview{}, err
	}

	// Обновляем timestamp
	now := time.Now()
	manifest.LastAppliedAt = &now
	db.Save(&manifest)

	// Delete ToDelete items from library and remove from manifest
	var toDeleteItems []ServerModManifestItem
	db.Preload("ModAsset").Where("server_mod_manifest_id = ? AND to_delete = ?", manifest.ID, true).Find(&toDeleteItems)
	for _, item := range toDeleteItems {
		if err := DeleteModAsset(db, item.ModAssetID); err != nil {
			log.Printf("Warning: failed to delete mod %s from library: %v", item.ModAsset.Name, err)
		}
		// Remove manifest item explicitly
		db.Unscoped().Delete(&item)
	}

	log.Printf("Applied manifest for server %s: +%d -%d", serverID, len(preview.ToAdd), len(preview.ToRemove))
	return PreviewApply(db, serverID)
}

func sortModStates(states []ModDeployState) {
	sort.Slice(states, func(i, j int) bool {
		return states[i].Name < states[j].Name
	})
}
