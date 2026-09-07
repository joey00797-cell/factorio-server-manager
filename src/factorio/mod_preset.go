package factorio

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// PresetModEntry represents a single mod in a preset
type PresetModEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

// PresetFile is the JSON structure stored on disk
type PresetFile struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Mods        []PresetModEntry `json:"mods"`
}

// PresetSummary is returned in list responses (no mods)
type PresetSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ModCount    int    `json:"mod_count"`
}

func presetsDir(serverID string) string {
	serverObj, ok := GetServerManager().GetServer(serverID)
	if !ok {
		return ""
	}
	// Store presets alongside mod_packs dir, not inside it
	// to avoid conflict with legacy ModPack directory scanner
	return filepath.Join(filepath.Dir(serverObj.Paths.ModPackDir), "presets")
}

func presetPath(serverID, name string) string {
	return filepath.Join(presetsDir(serverID), name+".json")
}

// ListPresets returns summaries of all presets for a server
func ListPresets(serverID string) ([]PresetSummary, error) {
	dir := presetsDir(serverID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var summaries []PresetSummary
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var pf PresetFile
		if err := json.Unmarshal(data, &pf); err != nil {
			continue
		}
		summaries = append(summaries, PresetSummary{
			Name:        pf.Name,
			Description: pf.Description,
			ModCount:    len(pf.Mods),
		})
	}
	return summaries, nil
}

// GetPreset returns full preset with mods
func GetPreset(serverID, name string) (PresetFile, error) {
	data, err := os.ReadFile(presetPath(serverID, name))
	if err != nil {
		return PresetFile{}, fmt.Errorf("preset not found: %s", name)
	}
	var pf PresetFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return PresetFile{}, err
	}
	return pf, nil
}

// SavePresetFromManifest creates or updates a preset from current manifest
func SavePresetFromManifest(db *gorm.DB, serverID, name, description string) (PresetFile, error) {
	manifest, err := EnsureManifest(db, serverID)
	if err != nil {
		return PresetFile{}, err
	}

	var mods []PresetModEntry
	for _, item := range manifest.Items {
		if item.ToDelete {
			continue
		}
		mods = append(mods, PresetModEntry{
			Name:    item.ModAsset.Name,
			Version: item.ModAsset.Version,
			Enabled: item.Enabled,
		})
	}

	pf := PresetFile{
		Name:        name,
		Description: description,
		Mods:        mods,
	}

	if err := savePresetFile(serverID, pf); err != nil {
		return PresetFile{}, err
	}
	return pf, nil
}

// LoadPresetToManifest applies a preset to the server manifest
func LoadPresetToManifest(db *gorm.DB, serverID, name string) (ManifestResult, error) {
	pf, err := GetPreset(serverID, name)
	if err != nil {
		return ManifestResult{}, err
	}

	// Build new manifest items from preset
	var newItems []struct {
		AssetID  uint `json:"asset_id"`
		Enabled  bool `json:"enabled"`
		ToDelete bool `json:"to_delete"`
	}

	for _, mod := range pf.Mods {
		// Find asset in library by name and version
		var asset ModAsset
		err := db.Where("name = ? AND version = ?", mod.Name, mod.Version).First(&asset).Error
		if err != nil {
			log.Printf("LoadPresetToManifest: mod %s %s not in library, skipping", mod.Name, mod.Version)
			continue
		}
		newItems = append(newItems, struct {
			AssetID  uint `json:"asset_id"`
			Enabled  bool `json:"enabled"`
			ToDelete bool `json:"to_delete"`
		}{AssetID: asset.ID, Enabled: mod.Enabled, ToDelete: false})
	}

	return UpdateManifestItems(db, serverID, newItems)
}

// DeletePreset removes a preset file
func DeletePreset(serverID, name string) error {
	path := presetPath(serverID, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AddModToPreset adds or updates a mod in an existing preset
func AddModToPreset(db *gorm.DB, serverID, presetName string, assetID uint) (PresetFile, error) {
	pf, err := GetPreset(serverID, presetName)
	if err != nil {
		return PresetFile{}, err
	}

	var asset ModAsset
	if err := db.First(&asset, assetID).Error; err != nil {
		return PresetFile{}, fmt.Errorf("asset not found: %d", assetID)
	}

	// Update if exists, add if not
	found := false
	for i, m := range pf.Mods {
		if m.Name == asset.Name {
			pf.Mods[i].Version = asset.Version
			pf.Mods[i].Enabled = true
			found = true
			break
		}
	}
	if !found {
		pf.Mods = append(pf.Mods, PresetModEntry{
			Name:    asset.Name,
			Version: asset.Version,
			Enabled: true,
		})
	}

	if err := savePresetFile(serverID, pf); err != nil {
		return PresetFile{}, err
	}
	return pf, nil
}

// RemoveModFromPreset removes a mod from a preset
func RemoveModFromPreset(serverID, presetName, modName string) (PresetFile, error) {
	pf, err := GetPreset(serverID, presetName)
	if err != nil {
		return PresetFile{}, err
	}

	var filtered []PresetModEntry
	for _, m := range pf.Mods {
		if m.Name != modName {
			filtered = append(filtered, m)
		}
	}
	pf.Mods = filtered

	if err := savePresetFile(serverID, pf); err != nil {
		return PresetFile{}, err
	}
	return pf, nil
}

func savePresetFile(serverID string, pf PresetFile) error {
	dir := presetsDir(serverID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Sanitize name for filename
	safeName := strings.ReplaceAll(pf.Name, "/", "_")
	safeName = strings.ReplaceAll(safeName, "..", "_")
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, safeName+".json"), data, 0644)
}
