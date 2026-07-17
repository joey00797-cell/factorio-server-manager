package factorio

import (
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ModAsset — физический файл мода в общей библиотеке
type ModAsset struct {
	gorm.Model
	Name            string `json:"name" gorm:"uniqueIndex:idx_mod_assets_name_version;not null"`
	Title           string `json:"title"`
	Author          string `json:"author"`
	Version         string `json:"version" gorm:"uniqueIndex:idx_mod_assets_name_version;not null"`
	FactorioVersion string `json:"factorio_version"`
	FileName        string `json:"file_name"`
	ArtifactPath    string `json:"artifact_path" gorm:"not null"`
	SourceType      string `json:"source_type"` // "upload" | "portal"
	SourceRef       string `json:"source_ref"`
	DependenciesRaw string `json:"-" gorm:"column:dependencies_json"`
}

func (a *ModAsset) Dependencies() []string {
	if a.DependenciesRaw == "" {
		return nil
	}
	var deps []string
	_ = json.Unmarshal([]byte(a.DependenciesRaw), &deps)
	return deps
}

func (a *ModAsset) SetDependencies(deps []string) {
	b, _ := json.Marshal(deps)
	a.DependenciesRaw = string(b)
}

// ModPreset — именованный набор модов
type ModPreset struct {
	gorm.Model
	Name        string         `json:"name" gorm:"uniqueIndex;not null"`
	Description string         `json:"description"`
	Items       []ModPresetItem `json:"items" gorm:"constraint:OnDelete:CASCADE;"`
}

type ModPresetItem struct {
	gorm.Model
	ModPresetID uint     `json:"mod_preset_id" gorm:"uniqueIndex:idx_mod_preset_asset"`
	ModAssetID  uint     `json:"mod_asset_id" gorm:"uniqueIndex:idx_mod_preset_asset"`
	ModAsset    ModAsset `json:"mod_asset"`
	Enabled     bool     `json:"enabled"`
}

// ServerModManifest — желаемое состояние модов для конкретного сервера
type ServerModManifest struct {
	gorm.Model
	ServerID        string               `json:"server_id" gorm:"uniqueIndex;not null"`
	SettingsData    []byte               `json:"-"`
	SettingsSource  string               `json:"settings_source"` // "preserve" | "replace"
	LastAppliedAt   *time.Time           `json:"last_applied_at"`
	Items           []ServerModManifestItem `json:"items" gorm:"constraint:OnDelete:CASCADE;"`
}

type ServerModManifestItem struct {
	gorm.Model
	ServerModManifestID uint     `json:"server_mod_manifest_id" gorm:"uniqueIndex:idx_server_manifest_asset"`
	ModAssetID          uint     `json:"mod_asset_id" gorm:"uniqueIndex:idx_server_manifest_asset"`
	ModAsset            ModAsset `json:"mod_asset"`
	Enabled             bool     `json:"enabled"`
	ToDelete            bool     `json:"to_delete" gorm:"default:false"`
}

// ModDeployState — состояние одного мода (deployed или desired)
type ModDeployState struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	AssetID *uint  `json:"asset_id,omitempty"`
	IsDLC   bool   `json:"is_dlc,omitempty"`
}

// ModPreviewIssue — проблема найденная при валидации
type ModPreviewIssue struct {
	Level   string `json:"level"` // "error" | "warning"
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ModApplyPreview — diff между желаемым и реальным состоянием
type ModApplyPreview struct {
	ServerID        string           `json:"server_id"`
	CanApply        bool             `json:"can_apply"`
	NeedsServerStop bool             `json:"needs_server_stop"`
	ToAdd           []ModDeployState `json:"to_add"`
	ToRemove        []ModDeployState `json:"to_remove"`
	ToEnable        []ModDeployState `json:"to_enable"`
	ToDisable       []ModDeployState `json:"to_disable"`
	Deployed        []ModDeployState `json:"deployed"`
	Desired         []ModDeployState `json:"desired"`
	Issues          []ModPreviewIssue `json:"issues"`
}

// Result типы для API
type ModAssetResult struct {
	ID              uint     `json:"id"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Author          string   `json:"author"`
	Version         string   `json:"version"`
	FactorioVersion string   `json:"factorio_version"`
	FileName        string   `json:"file_name"`
	SourceType      string   `json:"source_type"`
	Dependencies    []string `json:"dependencies"`
}

type ManifestItemResult struct {
	ID       uint           `json:"id"`
	AssetID  uint           `json:"asset_id"`
	Enabled  bool           `json:"enabled"`
	ToDelete bool           `json:"to_delete"`
	Asset    ModAssetResult `json:"asset"`
}

type ManifestResult struct {
	ServerID       string               `json:"server_id"`
	SettingsSource string               `json:"settings_source"`
	LastAppliedAt  *time.Time           `json:"last_applied_at"`
	Items          []ManifestItemResult `json:"items"`
}

type PresetResult struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Items       []ManifestItemResult `json:"items"`
}

func NewModAssetResult(a ModAsset) ModAssetResult {
	return ModAssetResult{
		ID:              a.ID,
		Name:            a.Name,
		Title:           a.Title,
		Author:          a.Author,
		Version:         a.Version,
		FactorioVersion: a.FactorioVersion,
		FileName:        a.FileName,
		SourceType:      a.SourceType,
		Dependencies:    a.Dependencies(),
	}
}

func newManifestItemResult(item ServerModManifestItem) ManifestItemResult {
	return ManifestItemResult{
		ID:       item.ID,
		AssetID:  item.ModAssetID,
		Enabled:  item.Enabled,
		ToDelete: item.ToDelete,
		Asset:    NewModAssetResult(item.ModAsset),
	}
}

func trimDepPrefix(dep, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(dep, prefix))
}
