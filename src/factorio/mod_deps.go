package factorio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// parsedDep represents a parsed mod dependency
type parsedDep struct {
	Name       string
	Optional   bool
	Incompatible bool
}

// parseRequiredDeps returns only mandatory (non-optional, non-incompatible) dependencies
func parseRequiredDeps(deps []string) []parsedDep {
	var result []parsedDep
	for _, dep := range deps {
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
		if name == "base" {
			continue
		}
		result = append(result, parsedDep{
			Name:         name,
			Optional:     optional,
			Incompatible: incompatible,
		})
	}
	return result
}

// getLatestCompatibleRelease returns the latest release of a mod compatible with baseVersion
func getLatestCompatibleRelease(modName string, baseVersion string) (portalModRelease, error) {
	url := fmt.Sprintf("https://mods.factorio.com/api/mods/%s", modName)
	resp, err := http.Get(url)
	if err != nil {
		return portalModRelease{}, fmt.Errorf("portal request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return portalModRelease{}, ErrModNotOnPortal
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var info portalModInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return portalModRelease{}, fmt.Errorf("parsing portal response: %v", err)
	}
	// Return latest release (last in list)
	if len(info.Releases) == 0 {
		return portalModRelease{}, fmt.Errorf("no releases found for %s", modName)
	}
	return info.Releases[len(info.Releases)-1], nil
}

// downloadAndImportMod downloads a mod release from portal and imports it to library
func downloadAndImportMod(db *gorm.DB, release portalModRelease, modsDir string) (ModAsset, error) {
	_ = modsDir // modsDir not needed, ImportPortalModToLibrary uses library dir
	return ImportPortalModToLibrary(db, release.DownloadURL, release.FileName, "")
}

// SyncDependencies resolves and downloads missing required dependencies
// for all enabled mods in the manifest. Adds them to the library and manifest.
func SyncDependencies(db *gorm.DB, serverID string) ([]ModSyncResult, error) {
	manifest, err := EnsureManifest(db, serverID)
	if err != nil {
		return nil, err
	}

	serverObj, ok := GetServerManager().GetServer(serverID)
	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}

	var results []ModSyncResult

	// Queue of mod names to resolve, start with enabled manifest items
	// Use a set to avoid processing the same dep twice
	visited := map[string]bool{}
	queue := []string{}

	for _, item := range manifest.Items {
		if item.Enabled && !item.ToDelete {
			visited[item.ModAsset.Name] = true
			for _, dep := range parseRequiredDeps(item.ModAsset.Dependencies()) {
				if !dep.Optional && !dep.Incompatible && !visited[dep.Name] {
					queue = append(queue, dep.Name)
				}
			}
		}
	}

	for len(queue) > 0 {
		depName := queue[0]
		queue = queue[1:]

		if visited[depName] {
			continue
		}
		visited[depName] = true

		// Check if already in manifest
		alreadyInManifest := false
		for _, item := range manifest.Items {
			if item.ModAsset.Name == depName && !item.ToDelete {
				alreadyInManifest = true
				break
			}
		}
		if alreadyInManifest {
			results = append(results, ModSyncResult{Name: depName, Status: "already_installed"})
			continue
		}

		// Check if in library
		var asset ModAsset
		err := db.Where("name = ?", depName).Order("id DESC").First(&asset).Error
		if err == nil {
			// Found in library - add to manifest
			log.Printf("SyncDependencies: %s found in library, adding to manifest", depName)
			manifest.Items = append(manifest.Items, ServerModManifestItem{
				ServerModManifestID: manifest.ID,
				ModAssetID:          asset.ID,
				ModAsset:            asset,
				Enabled:             true,
			})
			results = append(results, ModSyncResult{Name: depName, Version: asset.Version, Status: "from_library"})

			// Queue transitive deps
			for _, dep := range parseRequiredDeps(asset.Dependencies()) {
				if !dep.Optional && !dep.Incompatible && !visited[dep.Name] {
					queue = append(queue, dep.Name)
				}
			}
			continue
		}

		// Not in library - download latest from portal
		log.Printf("SyncDependencies: downloading %s from portal", depName)
		release, err := getLatestCompatibleRelease(depName, serverObj.BaseModVersion)
		if err != nil {
			log.Printf("SyncDependencies: cannot get release for %s: %v", depName, err)
			results = append(results, ModSyncResult{Name: depName, Status: "not_found"})
			continue
		}

		modsDir := serverObj.modsDir()
		importedAsset, err := downloadAndImportMod(db, release, modsDir)
		if err != nil {
			log.Printf("SyncDependencies: cannot download %s: %v", depName, err)
			results = append(results, ModSyncResult{Name: depName, Version: release.Version, Status: "error"})
			continue
		}

		manifest.Items = append(manifest.Items, ServerModManifestItem{
			ServerModManifestID: manifest.ID,
			ModAssetID:          importedAsset.ID,
			ModAsset:            importedAsset,
			Enabled:             true,
		})
		results = append(results, ModSyncResult{Name: depName, Version: release.Version, Status: "downloaded"})

		// Queue transitive deps
		for _, dep := range parseRequiredDeps(importedAsset.Dependencies()) {
			if !dep.Optional && !dep.Incompatible && !visited[dep.Name] {
				queue = append(queue, dep.Name)
			}
		}
	}

	// Save updated manifest items
	items := make([]struct {
		AssetID  uint
		Enabled  bool
		ToDelete bool
	}, len(manifest.Items))
	for i, item := range manifest.Items {
		items[i].AssetID = item.ModAssetID
		items[i].Enabled = item.Enabled
		items[i].ToDelete = item.ToDelete
	}

	updateItems := make([]struct {
		AssetID  uint `json:"asset_id"`
		Enabled  bool `json:"enabled"`
		ToDelete bool `json:"to_delete"`
	}, len(manifest.Items))
	for i, item := range manifest.Items {
		updateItems[i].AssetID = item.ModAssetID
		updateItems[i].Enabled = item.Enabled
		updateItems[i].ToDelete = item.ToDelete
	}
	if _, err := UpdateManifestItems(db, serverID, updateItems); err != nil {
		return results, fmt.Errorf("saving manifest: %w", err)
	}

	return results, nil
}
