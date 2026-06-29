package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

func GetModSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}
	path := server.Paths.ModSettingsDat
	if path == "" {
		path = filepath.Join(serverModsDir(server), "mod-settings.dat")
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		resp = map[string]interface{}{
			"exists":   false,
			"size":     0,
			"base64":   "",
			"settings": []interface{}{},
			"note":     "mod-settings.dat does not exist yet for this server",
		}
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error reading mod settings: %s", err)
		return
	}
	resp = map[string]interface{}{
		"exists":   true,
		"size":     len(raw),
		"base64":   base64.StdEncoding.EncodeToString(raw),
		"settings": []interface{}{},
		"note":     "raw mod-settings.dat editor; typed mod option decoding is not available for this file yet",
	}
}

func UpdateModSettingsHandler(w http.ResponseWriter, r *http.Request) {
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
		Base64 string `json:"base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = fmt.Sprintf("invalid mod settings request: %s", err)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(data.Base64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = fmt.Sprintf("invalid base64 mod settings: %s", err)
		return
	}
	path := server.Paths.ModSettingsDat
	if path == "" {
		path = filepath.Join(serverModsDir(server), "mod-settings.dat")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error saving mod settings: %s", err)
		return
	}
	if server.Running {
		server.PendingRestart = true
		if manager := factorio.GetServerManager(); manager != nil {
			_ = manager.UpdateServer(server)
		}
	}
	resp = map[string]interface{}{"saved": true, "size": len(raw)}
}
