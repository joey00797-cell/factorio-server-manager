package factorio

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type latestReleaseChannel struct {
	Headless string `json:"headless"`
}

func resolveFactorioVersion(requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = "stable"
	}
	if requested != "stable" && requested != "experimental" {
		return sanitizeVersionName(requested)
	}

	resp, err := http.Get("https://factorio.com/api/latest-releases")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("latest releases returned HTTP %d", resp.StatusCode)
	}

	var releases map[string]latestReleaseChannel
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	channel, ok := releases[requested]
	if !ok || strings.TrimSpace(channel.Headless) == "" {
		return "", fmt.Errorf("could not resolve Factorio %s headless version", requested)
	}
	return sanitizeVersionName(channel.Headless)
}

func sanitizeVersionName(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" || strings.Contains(version, "..") || strings.ContainsAny(version, `/\`) {
		return "", fmt.Errorf("invalid Factorio version %q", version)
	}
	return version, nil
}

func (m *ServerManager) ListInstalledVersions() []string {
	entries, err := os.ReadDir(m.versionDir)
	if err != nil {
		return []string{}
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		binary := filepath.Join(m.versionDir, e.Name(), "bin", "x64", "factorio")
		if _, err := os.Stat(binary); err == nil {
			versions = append(versions, e.Name())
		}
	}
	return versions
}

func (m *ServerManager) ListDownloadedVersions() []string {
	entries, err := os.ReadDir(m.downloadDir)
	if err != nil {
		return []string{}
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		var ver string
		if _, err := fmt.Sscanf(name, "factorio_%s", &ver); err == nil {
			ver = strings.TrimSuffix(ver, ".tar.xz")
			versions = append(versions, ver)
		}
	}
	return versions
}

func (m *ServerManager) DeleteInstalledVersion(version string) error {
	clean := filepath.Clean(version)
	if clean != version || strings.Contains(clean, "/") || strings.Contains(clean, "..") {
		return fmt.Errorf("invalid version name: %s", version)
	}
	m.mu.RLock()
	for _, srv := range m.catalog.Servers {
		if filepath.Base(srv.Paths.VersionDir) == clean {
			m.mu.RUnlock()
			return fmt.Errorf("version %s is in use by server %s", version, srv.ID)
		}
	}
	m.mu.RUnlock()
	versionDir := filepath.Join(m.versionDir, clean)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", version)
	}
	return os.RemoveAll(versionDir)
}

func (m *ServerManager) DeleteDownload(version string) error {
	clean := filepath.Clean(version)
	if clean != version || strings.Contains(clean, "/") || strings.Contains(clean, "..") {
		return fmt.Errorf("invalid version name: %s", version)
	}
	archive := filepath.Join(m.downloadDir, fmt.Sprintf("factorio_%s.tar.xz", clean))
	if _, err := os.Stat(archive); os.IsNotExist(err) {
		return fmt.Errorf("archive not found: %s", archive)
	}
	return os.Remove(archive)
}

func (m *ServerManager) EnsureVersionInstalled(requested string) (string, error) {
	resolved, err := resolveFactorioVersion(requested)
	if err != nil {
		return "", err
	}

	m.installMu.Lock()
	defer m.installMu.Unlock()

	versionDir := filepath.Join(m.versionDir, resolved)
	binary := filepath.Join(versionDir, "bin", "x64", "factorio")
	if _, err := os.Stat(binary); err == nil {
		return resolved, nil
	}

	if err := os.MkdirAll(m.downloadDir, 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(m.downloadDir, fmt.Sprintf("factorio_%s.tar.xz", resolved))
	url := fmt.Sprintf("https://www.factorio.com/get-download/%s/headless/linux64", resolved)
	log.Printf("Downloading Factorio %s from %s", resolved, url)

	out, err := os.Create(archive)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		out.Close()
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}

	partialDir := versionDir + ".partial"
	_ = os.RemoveAll(partialDir)
	if err := os.MkdirAll(partialDir, 0755); err != nil {
		return "", err
	}
	log.Printf("Extracting Factorio %s to %s", resolved, versionDir)
	cmd := exec.Command("tar", "-xf", archive, "-C", partialDir, "--strip-components=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(partialDir)
		return "", fmt.Errorf("extracting Factorio %s failed: %v: %s", resolved, err, strings.TrimSpace(string(output)))
	}
	_ = os.RemoveAll(versionDir)
	if err := os.Rename(partialDir, versionDir); err != nil {
		_ = os.RemoveAll(partialDir)
		return "", err
	}
	return resolved, nil
}

func (m *ServerManager) EnsureServerVersion(server *Server) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}
	requested := server.VersionChannel
	if strings.TrimSpace(requested) == "" {
		requested = server.VersionLabel
	}
	if strings.TrimSpace(requested) == "" && server.Paths.VersionDir != "" {
		requested = filepath.Base(server.Paths.VersionDir)
	}
	if strings.TrimSpace(requested) == "" || requested == "uninstalled" {
		requested = "stable"
	}
	server.VersionChannel = requested

	resolved, err := m.EnsureVersionInstalled(requested)
	if err != nil {
		return err
	}

	versionDir := filepath.Join(m.versionDir, resolved)
	binary := filepath.Join(versionDir, "bin", "x64", "factorio")
	if _, err := os.Stat(binary); err != nil {
		return fmt.Errorf("Factorio %s binary is unavailable at %s: %w", resolved, binary, err)
	}

	if server.VersionLabel == resolved && samePath(server.Paths.VersionDir, versionDir) {
		return EnsureInstanceFiles(server.Paths)
	}

	instanceRoot := server.Paths.Root
	if strings.TrimSpace(instanceRoot) == "" {
		instanceRoot = filepath.Join(m.instanceDir, server.ID)
	}
	server.VersionLabel = resolved
	server.Paths = buildInstancePaths(instanceRoot, versionDir)
	if err := EnsureInstanceFiles(server.Paths); err != nil {
		return err
	}
	if err := server.loadMetadata(); err != nil {
		log.Printf("Server %s metadata warning: %v", server.ID, err)
	}
	return m.UpdateServer(server)
}

func EnsureInstanceFiles(paths InstancePaths) error {
	if err := createInstanceDirs(paths); err != nil {
		return err
	}
	if _, err := os.Stat(paths.SettingsFile); os.IsNotExist(err) {
		if err := os.WriteFile(paths.SettingsFile, []byte(getDefaultServerSettings()), 0644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(paths.AdminFile); os.IsNotExist(err) {
		if err := os.WriteFile(paths.AdminFile, []byte("[]"), 0644); err != nil {
			return err
		}
	}
	return EnsureInstanceConfig(paths)
}

func EnsureInstanceConfig(paths InstancePaths) error {
	config := map[string]map[string]string{
		"path": {
			"read-data":  filepath.Join(paths.VersionDir, "data"),
			"write-data": paths.Root,
		},
		"general": {},
	}

	if _, err := os.Stat(paths.ConfigFile); err == nil {
		existing, err := LoadConfig(paths.ConfigFile)
		if err != nil {
			log.Printf("Repairing unreadable config.ini at %s: %v", paths.ConfigFile, err)
		} else {
			config = existing
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if config["path"] == nil {
		config["path"] = map[string]string{}
	}
	config["path"]["read-data"] = filepath.Join(paths.VersionDir, "data")
	config["path"]["write-data"] = paths.Root
	if config["general"] == nil {
		config["general"] = map[string]string{}
	}

	return SaveConfig(paths.ConfigFile, config)
}
