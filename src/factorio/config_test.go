package factorio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveConfigWritesStrictFactorioIni(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.ini")
	readData := "/opt/factorio-server/versions/2.0.77/data"
	writeData := "/opt/factorio-server/instances/5"

	err := SaveConfig(configFile, map[string]map[string]string{
		"path": {
			"read-data":  readData,
			"write-data": writeData,
		},
		"general": {
			"locale": "auto",
		},
	})
	if err != nil {
		t.Fatalf("SaveConfig returned error: %s", err)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("reading saved config: %s", err)
	}
	content := string(data)
	if !strings.Contains(content, "read-data="+readData) {
		t.Fatalf("expected strict read-data entry, got:\n%s", content)
	}
	if !strings.Contains(content, "write-data="+writeData) {
		t.Fatalf("expected strict write-data entry, got:\n%s", content)
	}
	if strings.Contains(content, "read-data ") || strings.Contains(content, "= "+readData) {
		t.Fatalf("config contains Factorio-unsafe padding:\n%s", content)
	}
}

func TestEnsureInstanceConfigRepairsPaddedPathConfig(t *testing.T) {
	tempDir := t.TempDir()
	paths := InstancePaths{
		Root:       filepath.Join(tempDir, "instances", "5"),
		VersionDir: filepath.Join(tempDir, "versions", "2.0.77"),
		ConfigFile: filepath.Join(tempDir, "instances", "5", "config", "config.ini"),
	}
	if err := os.MkdirAll(filepath.Dir(paths.ConfigFile), 0755); err != nil {
		t.Fatalf("creating config dir: %s", err)
	}

	err := os.WriteFile(paths.ConfigFile, []byte(`[path]
read-data  = __PATH__system-read-data__
write-data = __PATH__system-write-data__

[general]
locale = auto
`), 0644)
	if err != nil {
		t.Fatalf("writing padded config: %s", err)
	}

	if err := EnsureInstanceConfig(paths); err != nil {
		t.Fatalf("EnsureInstanceConfig returned error: %s", err)
	}

	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		t.Fatalf("reading repaired config: %s", err)
	}
	content := string(data)
	expectedReadData := filepath.Join(paths.VersionDir, "data")
	if !strings.Contains(content, "read-data="+expectedReadData) {
		t.Fatalf("expected repaired read-data entry, got:\n%s", content)
	}
	if !strings.Contains(content, "write-data="+paths.Root) {
		t.Fatalf("expected repaired write-data entry, got:\n%s", content)
	}
	if strings.Contains(content, "__PATH__system") || strings.Contains(content, "read-data ") || strings.Contains(content, "= "+expectedReadData) {
		t.Fatalf("config still contains Factorio-unsafe path data:\n%s", content)
	}
}

func TestEnsureInstanceFilesCreatesOwnedSaveDirAndWriteData(t *testing.T) {
	tempDir := t.TempDir()
	paths := buildInstancePaths(
		filepath.Join(tempDir, "instances", "7"),
		filepath.Join(tempDir, "versions", "2.0.77"),
	)

	if err := EnsureInstanceFiles(paths); err != nil {
		t.Fatalf("EnsureInstanceFiles returned error: %s", err)
	}

	if info, err := os.Stat(paths.SavesDir); err != nil || !info.IsDir() {
		t.Fatalf("expected owned saves directory at %s, stat err: %v", paths.SavesDir, err)
	}

	config, err := LoadConfig(paths.ConfigFile)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %s", err)
	}
	if got := config["path"]["write-data"]; got != paths.Root {
		t.Fatalf("expected write-data %q, got %q", paths.Root, got)
	}
	if got := config["path"]["read-data"]; got != filepath.Join(paths.VersionDir, "data") {
		t.Fatalf("expected read-data for server version, got %q", got)
	}
}
