package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

type wipeRequest struct {
	Saves   bool `json:"saves"`
	Mods    bool `json:"mods"`
	Config  bool `json:"config"`
	Logs    bool `json:"logs"`
	Backups bool `json:"backups"`
	Full    bool `json:"full"`
}

// clearDirContents removes everything inside dir, keeping dir itself.
// dir must be strictly inside root. Entries named in keep are skipped.
func clearDirContents(root, dir string, keep ...string) (int, error) {
	cleanRoot := filepath.Clean(root)
	cleanDir := filepath.Clean(dir)
	if root == "" || dir == "" || !strings.HasPrefix(cleanDir, cleanRoot+string(os.PathSeparator)) {
		return 0, fmt.Errorf("refusing to wipe %q", dir)
	}
	return clearEntries(cleanDir, keep...)
}

func clearEntries(dir string, keep ...string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
outer:
	for _, e := range entries {
		for _, k := range keep {
			if e.Name() == k {
				continue outer
			}
		}
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// wipeBackups clears the backups dir of one server. The dir name must equal
// the server id, so a wrong path can never wipe something else.
func wipeBackups(dir, serverID, scheduleFile string, removeDir bool) (int, error) {
	if dir == "" || serverID == "" || filepath.Base(filepath.Clean(dir)) != serverID {
		return 0, fmt.Errorf("refusing to wipe backups dir %q", dir)
	}
	if removeDir {
		n, err := clearEntries(filepath.Clean(dir))
		if err != nil {
			return n, err
		}
		return n, os.RemoveAll(filepath.Clean(dir))
	}
	return clearEntries(filepath.Clean(dir), filepath.Base(scheduleFile))
}

// WipeServerInstanceHandler clears selected parts of one server instance,
// or deletes the server completely. The mod library is never touched.
// Backups are touched only when explicitly requested.
func WipeServerInstanceHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp = "server not found"
		return
	}
	if server.GetRunning() {
		w.WriteHeader(http.StatusConflict)
		resp = "server must be stopped before wipe"
		return
	}

	var req wipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		resp = err.Error()
		return
	}

	serverID := server.ID
	instRoot := server.Paths.Root
	paths := server.Paths
	result := map[string]int{}
	fail := func(part string, err error) {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("wipe %s failed: %v", part, err)
	}

	if req.Full {
		if instRoot == "" || serverID == "" {
			fail("server", fmt.Errorf("empty instance path"))
			return
		}
		manager := factorio.GetServerManager()
		if manager == nil {
			fail("server", fmt.Errorf("server manager is not initialized"))
			return
		}
		if err := manager.DeleteServer(serverID); err != nil {
			fail("server", err)
			return
		}
		if err := os.RemoveAll(instRoot); err != nil {
			fail("instance", err)
			return
		}
		result["instance"] = 1

		// hard delete: server_id has a unique index and ids are reused
		db := GetDB()
		var mf factorio.ServerModManifest
		if err := db.Unscoped().Where("server_id = ?", serverID).First(&mf).Error; err == nil {
			db.Unscoped().Where("server_mod_manifest_id = ?", mf.ID).Delete(&factorio.ServerModManifestItem{})
			db.Unscoped().Delete(&mf)
			result["manifest"] = 1
		}
		if req.Backups {
			n, err := wipeBackups(paths.BackupsDir, serverID, paths.BackupScheduleFile, true)
			if err != nil {
				fail("backups", err)
				return
			}
			result["backups"] = n
		}
		fmt.Printf("[FSM] full delete server %s: %v\n", serverID, result)
		resp = result
		return
	}

	if req.Saves {
		n, err := clearDirContents(instRoot, paths.SavesDir)
		if err != nil {
			fail("saves", err)
			return
		}
		result["saves"] = n
	}
	if req.Mods {
		n, err := clearDirContents(instRoot, paths.ModsDir)
		if err != nil {
			fail("mods", err)
			return
		}
		result["mods"] = n
		if manifest, err := factorio.EnsureManifest(GetDB(), serverID); err == nil {
			GetDB().Unscoped().Where("server_mod_manifest_id = ?", manifest.ID).Delete(&factorio.ServerModManifestItem{})
		}
	}
	if req.Config {
		n, err := clearDirContents(instRoot, paths.ConfigDir)
		if err != nil {
			fail("config", err)
			return
		}
		result["config"] = n
	}
	if req.Logs {
		n, err := clearDirContents(instRoot, paths.LogsDir)
		if err != nil {
			fail("logs", err)
			return
		}
		result["logs"] = n
	}
	if req.Backups {
		n, err := wipeBackups(paths.BackupsDir, serverID, paths.BackupScheduleFile, false)
		if err != nil {
			fail("backups", err)
			return
		}
		result["backups"] = n
	}

	fmt.Printf("[FSM] wipe server %s: %v\n", serverID, result)
	resp = result
}
