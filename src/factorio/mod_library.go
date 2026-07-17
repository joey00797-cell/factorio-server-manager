package factorio

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"gorm.io/gorm"
)

const modLibrarySubdir = "mod-library"

var dlcMods = []ModAsset{
	{Name: "space-age", Title: "Space Age", SourceType: "dlc", FileName: "", ArtifactPath: ""},
	{Name: "elevated-rails", Title: "Elevated Rails", SourceType: "dlc", FileName: "", ArtifactPath: ""},
	{Name: "quality", Title: "Quality", SourceType: "dlc", FileName: "", ArtifactPath: ""},
}

// EnsureDLCAssets создаёт записи DLC модов в библиотеке если их нет
func EnsureDLCAssets(db *gorm.DB) error {
	for _, dlc := range dlcMods {
		var existing ModAsset
		err := db.Where("name = ? AND source_type = ?", dlc.Name, "dlc").First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			dlcCopy := dlc
			if err := db.Create(&dlcCopy).Error; err != nil {
				return err
			}
			log.Printf("Created DLC asset: %s", dlc.Name)
		}
	}
	return nil
}

func GetModLibraryDir() string {
	config := bootstrap.GetConfig()
	return filepath.Join(config.FactorioDir, modLibrarySubdir)
}

func ensureModLibraryDir() error {
	return os.MkdirAll(GetModLibraryDir(), 0755)
}

// isModpackZip checks if zip contains mod-list.json (Factorio 2.0 modpack)
func isModpackZip(path string) bool {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "mod-list.json" || strings.HasSuffix(f.Name, "/mod-list.json") {
			return true
		}
	}
	return false
}

// ImportModpackZip imports all .zip mods from a Factorio 2.0 modpack archive
func ImportModpackZip(db *gorm.DB, packPath string) ([]ModAsset, error) {
	r, err := zip.OpenReader(packPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open modpack: %w", err)
	}
	defer r.Close()

	var imported []ModAsset
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".zip") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			log.Printf("Warning: failed to open mod entry %s: %v", f.Name, err)
			continue
		}
		baseName := filepath.Base(f.Name)
		asset, err := ImportModZipToLibrary(db, rc, baseName, "upload", "")
		rc.Close()
		if err != nil {
			log.Printf("Warning: failed to import mod %s from modpack: %v", baseName, err)
			continue
		}
		imported = append(imported, asset)
	}
	return imported, nil
}

// readModInfoFromZip читает info.json из zip архива мода
func readModInfoFromZip(path string) (map[string]interface{}, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/info.json") || f.Name == "info.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			var info map[string]interface{}
			if err := json.NewDecoder(rc).Decode(&info); err != nil {
				return nil, err
			}
			return info, nil
		}
	}
	return nil, fmt.Errorf("info.json not found in mod zip")
}

func assetFromInfo(info map[string]interface{}, fileName, artifactPath, sourceType, sourceRef string) (ModAsset, error) {
	getStr := func(key string) string {
		if v, ok := info[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	name := getStr("name")
	version := getStr("version")
	if name == "" || version == "" {
		return ModAsset{}, fmt.Errorf("mod info.json missing name or version")
	}

	asset := ModAsset{
		Name:            name,
		Title:           getStr("title"),
		Author:          getStr("author"),
		Version:         version,
		FactorioVersion: getStr("factorio_version"),
		FileName:        fileName,
		ArtifactPath:    artifactPath,
		SourceType:      sourceType,
		SourceRef:       sourceRef,
	}

	// Зависимости
	if deps, ok := info["dependencies"]; ok {
		switch v := deps.(type) {
		case []interface{}:
			var depStrs []string
			for _, d := range v {
				if s, ok := d.(string); ok {
					depStrs = append(depStrs, s)
				}
			}
			asset.SetDependencies(depStrs)
		}
	}

	return asset, nil
}

// ImportModZipToLibrary импортирует zip файл в библиотеку
func ImportModZipToLibrary(db *gorm.DB, reader io.Reader, fileName, sourceType, sourceRef string) (ModAsset, error) {
	if err := ensureModLibraryDir(); err != nil {
		return ModAsset{}, err
	}

	// Сохраняем во временный файл
	tmpPath := filepath.Join(GetModLibraryDir(), "tmp_"+fileName)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return ModAsset{}, err
	}
	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return ModAsset{}, err
	}
	tmpFile.Close()

	// Читаем info.json
	info, err := readModInfoFromZip(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return ModAsset{}, fmt.Errorf("invalid mod zip: %w", err)
	}

	asset, err := assetFromInfo(info, fileName, "", sourceType, sourceRef)
	if err != nil {
		os.Remove(tmpPath)
		return ModAsset{}, err
	}

	// Проверяем нет ли уже такого мода
	var existing ModAsset
	if err := db.Where("name = ? AND version = ?", asset.Name, asset.Version).First(&existing).Error; err == nil {
		os.Remove(tmpPath)
		// Fix empty ArtifactPath for previously imported mods
		if existing.ArtifactPath == "" {
			finalPath := filepath.Join(GetModLibraryDir(), fileName)
			if err := os.Rename(tmpPath, finalPath); err == nil {
				db.Model(&existing).Update("artifact_path", finalPath)
				existing.ArtifactPath = finalPath
			}
		}
		return existing, nil
	}

	// Перемещаем в библиотеку
	finalPath := filepath.Join(GetModLibraryDir(), fileName)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return ModAsset{}, err
	}
	asset.ArtifactPath = finalPath

	if err := db.Create(&asset).Error; err != nil {
		os.Remove(finalPath)
		return ModAsset{}, err
	}

	log.Printf("Imported mod to library: %s %s", asset.Name, asset.Version)
	return asset, nil
}

// ImportUploadedModToLibrary imports uploaded mod zip or Factorio 2.0 modpack
func ImportUploadedModToLibrary(db *gorm.DB, file multipart.File, header *multipart.FileHeader) ([]ModAsset, error) {
	if err := ensureModLibraryDir(); err != nil {
		return nil, err
	}

	// Save upload to temp file for inspection
	tmpPath := filepath.Join(GetModLibraryDir(), "tmp_upload_"+header.Filename)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, err
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if isModpackZip(tmpPath) {
		// Factorio 2.0 modpack: extract and import each mod zip
		return ImportModpackZip(db, tmpPath)
	}

	// Single mod zip
	f, err := os.Open(tmpPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	asset, err := ImportModZipToLibrary(db, f, header.Filename, "upload", "")
	if err != nil {
		return nil, err
	}
	return []ModAsset{asset}, nil
}

// ImportPortalModToLibrary скачивает мод с портала и импортирует в библиотеку
func ImportPortalModToLibrary(db *gorm.DB, downloadURL, fileName, modName string) (ModAsset, error) {
	if err := ensureModLibraryDir(); err != nil {
		return ModAsset{}, err
	}

	creds := Credentials{}
	if ok, err := creds.Load(); err != nil || !ok {
		return ModAsset{}, fmt.Errorf("not logged in to mod portal")
	}

	// Скачиваем файл
	fullURL := "https://mods.factorio.com" + downloadURL + "?username=" + creds.Username + "&token=" + creds.Userkey
	resp, err := http.Get(fullURL)
	if err != nil {
		return ModAsset{}, err
	}
	defer resp.Body.Close()

	return ImportModZipToLibrary(db, resp.Body, fileName, "portal", modName)
}

// ListModAssets возвращает все моды в библиотеке
func ListModAssets(db *gorm.DB) ([]ModAssetResult, error) {
	var assets []ModAsset
	if err := db.Order("name, version").Find(&assets).Error; err != nil {
		return nil, err
	}
	results := make([]ModAssetResult, len(assets))
	for i, a := range assets {
		results[i] = NewModAssetResult(a)
	}
	return results, nil
}

// DeleteModAsset удаляет мод из библиотеки
func DeleteModAsset(db *gorm.DB, assetID uint) error {
	var asset ModAsset
	if err := db.First(&asset, assetID).Error; err != nil {
		return fmt.Errorf("asset not found: %w", err)
	}
	if err := os.Remove(asset.ArtifactPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to delete mod file %s: %v", asset.ArtifactPath, err)
	}
	// Clean up orphaned manifest items
	db.Unscoped().Where("mod_asset_id = ?", asset.ID).Delete(&ServerModManifestItem{})
	return db.Unscoped().Delete(&asset).Error
}
