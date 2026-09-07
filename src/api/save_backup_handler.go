package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func GetSaveBackupScheduleHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	schedule, err := factorio.LoadSaveBackupSchedule(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = schedule
}

func UpdateSaveBackupScheduleHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	var schedule factorio.SaveBackupSchedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = err.Error()
		return
	}

	saved, err := factorio.SaveBackupScheduleConfig(server, schedule)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = saved
}

func RunSaveBackupHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	backups, err := factorio.RunSaveBackup(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}

	schedule, _ := factorio.LoadSaveBackupSchedule(server)
	factorio.PruneSaveBackups(server, schedule.Retention)

	resp = backups
}

func ListSaveBackupsHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	backups, err := factorio.ListSaveBackups(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = backups
}

func ListAllSaveBackupsHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	backups, err := factorio.ListAllSaveBackups(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = backups
}

func DeleteSaveBackupHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["name"]
	backups, err := factorio.ListSaveBackups(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}

	for _, b := range backups {
		if b.Name == name {
			if err := os.Remove(b.Path); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				resp = err.Error()
				return
			}
			resp = map[string]bool{"deleted": true}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	resp = "backup not found"
}

func DownloadSaveBackupHandler(w http.ResponseWriter, r *http.Request) {
	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := mux.Vars(r)["name"]
	backups, err := factorio.ListSaveBackups(server)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, b := range backups {
		if b.Name == name {
			w.Header().Set("Content-Disposition", "attachment; filename="+name)
			w.Header().Set("Content-Type", "application/octet-stream")
			http.ServeFile(w, r, b.Path)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func RestoreSaveBackupHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	// Source server (where backup lives)
	srcServer, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	// Optional target server from body
	var body struct {
		TargetServerID string `json:"target_server_id"`
		SourceServerID string `json:"source_server_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Source server override (for cross-server backups)
	if body.SourceServerID != "" && body.SourceServerID != srcServer.ID {
		manager := factorio.GetServerManager()
		if manager != nil {
			if s, exists := manager.GetServer(body.SourceServerID); exists {
				srcServer = s
			}
		}
	}

	dstServer := srcServer
	if body.TargetServerID != "" && body.TargetServerID != srcServer.ID {
		manager := factorio.GetServerManager()
		if manager != nil {
			if s, exists := manager.GetServer(body.TargetServerID); exists {
				dstServer = s
			}
		}
	}

	name := mux.Vars(r)["name"]
	backups, err := factorio.ListSaveBackups(srcServer)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}

	for _, b := range backups {
		if b.Name == name {
			origName := name
			if len(name) > 16 && name[8] == '_' && name[15] == '_' {
				origName = name[16:]
			}
			dstPath := filepath.Join(dstServer.Paths.SavesDir, origName)
			if err := factorio.RestoreSaveBackup(b.Path, dstPath); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				resp = err.Error()
				return
			}
			resp = map[string]string{"restored": origName}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	resp = "backup not found"
}

func TogglePinBackupHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["name"]
	pinned, err := factorio.TogglePinBackup(server, name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = map[string]bool{"pinned": pinned}
}

func RenameBackupHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}

	name := mux.Vars(r)["name"]
	var body struct {
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewName == "" {
		w.WriteHeader(http.StatusBadRequest)
		resp = "new_name required"
		return
	}

	if err := factorio.RenameBackup(server, name, body.NewName); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	resp = map[string]string{"name": body.NewName}
}
