package factorio

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SaveBackupSchedule holds backup configuration for a server
type SaveBackupSchedule struct {
	Enabled         bool      `json:"enabled"`
	IntervalMinutes int       `json:"interval_minutes"`
	Retention       int       `json:"retention"`
	LastRun         time.Time `json:"last_run,omitempty"`
	NextRun         time.Time `json:"next_run,omitempty"`
}

// SaveBackup represents a single backup entry
type SaveBackup struct {
	Name             string    `json:"name"`
	Size             int64     `json:"size"`
	CreatedAt        time.Time `json:"created_at"`
	Pinned           bool      `json:"pinned"`
	Path             string    `json:"-"`
	SourceServerID   string    `json:"source_server_id,omitempty"`
	SourceServerName string    `json:"source_server_name,omitempty"`
}

func defaultSaveBackupSchedule() SaveBackupSchedule {
	return SaveBackupSchedule{
		Enabled:         false,
		IntervalMinutes: 60,
		Retention:       5,
	}
}

func LoadSaveBackupSchedule(server *Server) (SaveBackupSchedule, error) {
	schedule := defaultSaveBackupSchedule()
	data, err := os.ReadFile(server.Paths.BackupScheduleFile)
	if os.IsNotExist(err) {
		return schedule, nil
	}
	if err != nil {
		return schedule, err
	}
	err = json.Unmarshal(data, &schedule)
	return schedule, err
}

func SaveBackupScheduleConfig(server *Server, schedule SaveBackupSchedule) (SaveBackupSchedule, error) {
	if schedule.IntervalMinutes <= 0 {
		schedule.IntervalMinutes = 60
	}
	if schedule.Retention <= 0 {
		schedule.Retention = 5
	}
	data, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return schedule, err
	}
	return schedule, os.WriteFile(server.Paths.BackupScheduleFile, data, 0664)
}

func ListSaveBackups(server *Server) ([]SaveBackup, error) {
	entries, err := os.ReadDir(server.Paths.BackupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SaveBackup{}, nil
		}
		return nil, err
	}
	pinned, _ := LoadPinnedBackups(server)
	var backups []SaveBackup
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == "pinned.json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, SaveBackup{
			Name:      e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Pinned:    pinned[e.Name()],
			Path:      filepath.Join(server.Paths.BackupsDir, e.Name()),
		})
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	return backups, nil
}

// ListAllSaveBackups returns backups from all servers, tagged with source server info
func ListAllSaveBackups(currentServer *Server) ([]SaveBackup, error) {
	manager := GetServerManager()
	if manager == nil {
		return ListSaveBackups(currentServer)
	}
	servers := manager.ListServers()
	var all []SaveBackup
	for _, srv := range servers {
		backups, err := ListSaveBackups(srv)
		if err != nil {
			continue
		}
		for i := range backups {
			backups[i].SourceServerID = srv.ID
			backups[i].SourceServerName = srv.Name
		}
		all = append(all, backups...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all, nil
}

func RunSaveBackup(server *Server) ([]SaveBackup, error) {
	saves, err := os.ReadDir(server.Paths.SavesDir)
	if err != nil {
		return nil, err
	}
	var created []SaveBackup
	existingBackups, _ := ListSaveBackups(server)
	for _, s := range saves {
		if s.IsDir() {
			continue
		}
		info, err := s.Info()
		if err != nil {
			continue
		}
		// Skip if last backup of this save has same size and mtime
		skip := false
		for _, b := range existingBackups {
			if len(b.Name) > 16 && b.Name[16:] == s.Name() {
				if b.Size == info.Size() {
					log.Printf("save_backup: skipping %s (no changes)", s.Name())
					skip = true
				}
				break
			}
		}
		if skip {
			continue
		}
		timestamp := time.Now().UTC().Format("20060102_150405")
		dstName := fmt.Sprintf("%s_%s", timestamp, s.Name())
		dstPath := filepath.Join(server.Paths.BackupsDir, dstName)
		if err := copyFile(filepath.Join(server.Paths.SavesDir, s.Name()), dstPath, 0664); err != nil {
			log.Printf("save_backup: error copying %s: %s", s.Name(), err)
			continue
		}
		created = append(created, SaveBackup{
			Name:      dstName,
			Size:      info.Size(),
			CreatedAt: time.Now(),
			Path:      dstPath,
		})
	}
	return created, nil
}

func PruneSaveBackups(server *Server, retention int) error {
	if retention <= 0 {
		return nil
	}
	backups, err := ListSaveBackups(server)
	if err != nil {
		return err
	}
	// Filter out pinned backups before pruning
	unpinned := []SaveBackup{}
	for _, b := range backups {
		if !b.Pinned {
			unpinned = append(unpinned, b)
		}
	}
	if len(unpinned) <= retention {
		return nil
	}
	for _, b := range unpinned[retention:] {
		if err := os.Remove(b.Path); err != nil {
			log.Printf("save_backup: error pruning %s: %s", b.Name, err)
		}
	}
	return nil
}

// Scheduler

var saveBackupSchedulerOnce sync.Once

func StartSaveBackupScheduler() {
	saveBackupSchedulerOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				manager := GetServerManager()
				if manager == nil {
					continue
				}
				for _, server := range manager.ListServers() {
					schedule, err := LoadSaveBackupSchedule(server)
					if err != nil {
						log.Printf("save_backup: error loading schedule for server %s: %s", server.ID, err)
						continue
					}
					if !schedule.Enabled {
						continue
					}
					if schedule.NextRun.IsZero() {
						schedule.NextRun = time.Now().UTC().Add(time.Duration(schedule.IntervalMinutes) * time.Minute)
						if _, err := SaveBackupScheduleConfig(server, schedule); err != nil {
							log.Printf("save_backup: error saving schedule: %s", err)
						}
						continue
					}
					if time.Now().UTC().Before(schedule.NextRun) {
						continue
					}
					backups, err := RunSaveBackup(server)
					if err != nil {
						log.Printf("save_backup: error running backup for server %s: %s", server.ID, err)
					} else {
						log.Printf("save_backup: created %d backups for server %s", len(backups), server.ID)
					}
					if err := PruneSaveBackups(server, schedule.Retention); err != nil {
						log.Printf("save_backup: error pruning backups for server %s: %s", server.ID, err)
					}
					schedule.LastRun = time.Now().UTC()
					schedule.NextRun = time.Now().UTC().Add(time.Duration(schedule.IntervalMinutes) * time.Minute)
					if _, err := SaveBackupScheduleConfig(server, schedule); err != nil {
						log.Printf("save_backup: error saving schedule after run: %s", err)
					}
				}
			}
		}()
	})
}

func RestoreSaveBackup(src, dst string) error {
	return copyFile(src, dst, 0664)
}

func pinnedPath(server *Server) string {
	return filepath.Join(server.Paths.BackupsDir, "pinned.json")
}

func LoadPinnedBackups(server *Server) (map[string]bool, error) {
	pinned := make(map[string]bool)
	data, err := os.ReadFile(pinnedPath(server))
	if os.IsNotExist(err) {
		return pinned, nil
	}
	if err != nil {
		return pinned, err
	}
	err = json.Unmarshal(data, &pinned)
	return pinned, err
}

func savePinnedBackups(server *Server, pinned map[string]bool) error {
	data, err := json.MarshalIndent(pinned, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pinnedPath(server), data, 0664)
}

func TogglePinBackup(server *Server, name string) (bool, error) {
	pinned, err := LoadPinnedBackups(server)
	if err != nil {
		return false, err
	}
	pinned[name] = !pinned[name]
	if !pinned[name] {
		delete(pinned, name)
	}
	return pinned[name], savePinnedBackups(server, pinned)
}

func RenameBackup(server *Server, oldName, newName string) error {
	oldPath := filepath.Join(server.Paths.BackupsDir, oldName)
	newPath := filepath.Join(server.Paths.BackupsDir, newName)
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	// Update pinned if needed
	pinned, _ := LoadPinnedBackups(server)
	if pinned[oldName] {
		delete(pinned, oldName)
		pinned[newName] = true
		savePinnedBackups(server, pinned)
	}
	return nil
}
