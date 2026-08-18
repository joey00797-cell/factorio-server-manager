package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func ListServersHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	manager := factorio.GetServerManager()
	if manager == nil {
		resp = []*factorio.Server{factorio.GetFactorioServer()}
		return
	}
	resp = manager.ListServers()
}

func PreviewNextServerHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = "server manager is not initialized"
		return
	}
	id, name, port := manager.PreviewNextServer()
	resp = map[string]interface{}{
		"id":   id,
		"name": name,
		"port": port,
	}
}

func CreateServerHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		BindIP    string `json:"bind_ip"`
		Port      int    `json:"port"`
		Autostart bool   `json:"autostart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = fmt.Sprintf("invalid server request: %s", err)
		return
	}

	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = "server manager is not initialized"
		return
	}
	server, err := manager.CreateServer(data.Name, data.Version, data.BindIP, data.Port, data.Autostart)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error creating server: %s", err)
		return
	}
	resp = server
}

func GetServerHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}
	resp = server
}

func UpdateServerHandler(w http.ResponseWriter, r *http.Request) {
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
		Name      *string `json:"name"`
		BindIP    *string `json:"bind_ip"`
		Port      *int    `json:"port"`
		Version   *string `json:"version"`
		Autostart         *bool   `json:"autostart"`
		WatchdogInterval *int    `json:"watchdog_interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = fmt.Sprintf("invalid server request: %s", err)
		return
	}
	manager := factorio.GetServerManager()
	if data.Port != nil {
		if *data.Port < 1 || *data.Port > 65535 {
			w.WriteHeader(http.StatusBadRequest)
			resp = "port must be between 1 and 65535"
			return
		}
		if manager != nil {
			for _, other := range manager.ListServers() {
				if other.ID != server.ID && other.Port == *data.Port {
					w.WriteHeader(http.StatusConflict)
					resp = fmt.Sprintf("port %d is already used by server %s", *data.Port, other.ID)
					return
				}
			}
		}
	}
	if data.Name != nil {
		server.Name = *data.Name
	}
	if data.BindIP != nil {
		if server.BindIP != *data.BindIP {
			server.PendingRestart = server.Running
		}
		server.BindIP = *data.BindIP
	}
	if data.Port != nil {
		if server.Port != *data.Port {
			server.PendingRestart = server.Running
		}
		server.Port = *data.Port
	}
	if data.Autostart != nil {
		server.Autostart = *data.Autostart
	}
	if data.WatchdogInterval != nil {
		if *data.WatchdogInterval < 0 {
			w.WriteHeader(http.StatusBadRequest)
			resp = "watchdog_interval must be >= 0"
			return
		}
		server.WatchdogInterval = *data.WatchdogInterval
	}
	if data.Version != nil {
		trimmed := strings.TrimSpace(*data.Version)
		if trimmed != "" && trimmed != server.VersionChannel {
			if server.Running {
				server.PendingRestart = true
				server.VersionChannel = trimmed
			} else if manager != nil {
				server.VersionChannel = trimmed
				if err := manager.EnsureServerVersion(server); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					resp = fmt.Sprintf("error installing Factorio %s: %s", trimmed, err)
					return
				}
			} else {
				server.VersionChannel = trimmed
			}
		}
	}
	if manager != nil {
		if err := manager.UpdateServer(server); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			resp = err.Error()
			return
		}
	}
	resp = server
}

func DeleteServerHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	id := mux.Vars(r)["serverID"]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		resp = "server id is required"
		return
	}
	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}
	if server.Running {
		w.WriteHeader(http.StatusLocked)
		resp = "server is running"
		return
	}

	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = "server manager is not initialized"
		return
	}
	if err := manager.DeleteServer(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = err.Error()
		return
	}
	_ = os.RemoveAll(server.Paths.Root)
	resp = true
}

func SaveServerHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}
	if !server.Running || server.Rcon == nil {
		w.WriteHeader(http.StatusConflict)
		resp = "server is not running or RCON is unavailable"
		return
	}
	reqID, err := server.Rcon.Write("/server-save")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("error saving server: %s", err)
		return
	}
	resp = map[string]interface{}{"saved": true, "request_id": reqID}
}
