package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

// GET /api/mods/library
func ListModAssetsHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	assets, err := factorio.ListModAssets(GetDB())
	if err != nil {
		resp = fmt.Sprintf("Error listing mod assets: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = assets
}

// POST /api/servers/{serverID}/mods/manifest/reset
func ResetManifestHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	serverID := vars["serverID"]

	result, err := factorio.ResetManifestToDeployed(GetDB(), serverID)
	if err != nil {
		resp = fmt.Sprintf("Error resetting manifest: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = result
}

// POST /api/mods/library/upload
func UploadModToLibraryHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	if err := r.ParseMultipartForm(200 << 20); err != nil {
		resp = "Failed to parse multipart form"
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("mod")
	if err != nil {
		resp = fmt.Sprintf("Error reading uploaded file: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	assets, err := factorio.ImportUploadedModToLibrary(GetDB(), file, header)
	if err != nil {
		resp = fmt.Sprintf("Error importing mod: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	results := make([]factorio.ModAssetResult, len(assets))
	for i, a := range assets {
		results[i] = factorio.NewModAssetResult(a)
	}
	resp = results
}

// POST /api/mods/library/portal-import
func ImportPortalModToLibraryHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		ModName     string `json:"mod_name"`
	}
	if _, err := ReadFromRequestBody(w, r, &data); err != nil {
		return
	}

	asset, err := factorio.ImportPortalModToLibrary(GetDB(), data.DownloadURL, data.FileName, data.ModName)
	if err != nil {
		resp = fmt.Sprintf("Error importing mod from portal: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = factorio.NewModAssetResult(asset)
}

// DELETE /api/mods/library/{id}
func DeleteModAssetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		resp = "Invalid asset ID"
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := factorio.DeleteModAsset(GetDB(), uint(id)); err != nil {
		resp = fmt.Sprintf("Error deleting mod asset: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = "deleted"
}

// GET /api/servers/{serverID}/mods/manifest
func GetManifestHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	serverID := vars["serverID"]

	manifest, err := factorio.GetManifest(GetDB(), serverID)
	if err != nil {
		resp = fmt.Sprintf("Error getting manifest: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = manifest
}

// PUT /api/servers/{serverID}/mods/manifest
func UpdateManifestHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	serverID := vars["serverID"]

	var data struct {
		Items []struct {
			AssetID  uint `json:"asset_id"`
			Enabled  bool `json:"enabled"`
			ToDelete bool `json:"to_delete"`
		} `json:"items"`
	}
	if _, err := ReadFromRequestBody(w, r, &data); err != nil {
		return
	}

	manifest, err := factorio.UpdateManifestItems(GetDB(), serverID, data.Items)
	if err != nil {
		resp = fmt.Sprintf("Error updating manifest: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = manifest
}

// GET /api/servers/{serverID}/mods/manifest/preview
func PreviewManifestHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	serverID := vars["serverID"]

	preview, err := factorio.PreviewApply(GetDB(), serverID)
	if err != nil {
		resp = fmt.Sprintf("Error previewing manifest: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = preview
}

// POST /api/servers/{serverID}/mods/manifest/apply
func ApplyManifestHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	serverID := vars["serverID"]

	result, err := factorio.ApplyManifest(GetDB(), serverID)
	if err != nil {
		resp = fmt.Sprintf("Error applying manifest: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp = result
}
