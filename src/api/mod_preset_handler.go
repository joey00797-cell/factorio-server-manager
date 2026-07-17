package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func ListPresetsHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	summaries, err := factorio.ListPresets(server.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error listing presets: %s", err)
		return
	}
	resp = summaries
}

func GetPresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["preset"]
	pf, err := factorio.GetPreset(server.ID, name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		resp = fmt.Sprintf("preset not found: %s", name)
		return
	}
	resp = pf
}

func SavePresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	var data struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	resp, err := ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	pf, err := factorio.SavePresetFromManifest(GetDB(), server.ID, data.Name, data.Description)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error saving preset: %s", err)
		return
	}
	resp = pf
}

func LoadPresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["preset"]
	result, err := factorio.LoadPresetToManifest(GetDB(), server.ID, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error loading preset: %s", err)
		return
	}
	resp = result
}

func DeletePresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["preset"]
	if err := factorio.DeletePreset(server.ID, name); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error deleting preset: %s", err)
		log.Println(resp)
		return
	}
	resp = true
}

func AddModToPresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["preset"]
	var data struct {
		AssetID uint `json:"asset_id"`
	}
	resp, err := ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	pf, err := factorio.AddModToPreset(GetDB(), server.ID, name, data.AssetID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error adding mod to preset: %s", err)
		return
	}
	resp = pf
}

func RemoveModFromPresetHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["preset"]
	var data struct {
		ModName string `json:"mod_name"`
	}
	resp, err := ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	pf, err := factorio.RemoveModFromPreset(server.ID, name, data.ModName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error removing mod from preset: %s", err)
		return
	}
	resp = pf
}
