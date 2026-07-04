package factorio

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type Save struct {
	Name    string    `json:"name"`
	LastMod time.Time `json:"last_mod"`
	Size    int64     `json:"size"`
}

func (s *Save) String() string {
	return s.Name
}

// Lists save files in factorio/saves
func ListSaves() (saves []Save, err error) {
	config := bootstrap.GetConfig()
	savesDir := config.FactorioSavesDir
	if manager := GetServerManager(); manager != nil {
		savesDir = manager.DefaultServer().savesDir()
	}
	return ListSavesInDir(savesDir)
}

func ListSavesInDir(savesDir string) (saves []Save, err error) {
	saves = []Save{}
	entries, err := os.ReadDir(savesDir)
	if os.IsNotExist(err) {
		return saves, nil
	}
	if err != nil {
		return saves, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := ValidateSaveName(name); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return saves, err
		}
		saves = append(saves, Save{
			name,
			info.ModTime(),
			info.Size(),
		})
	}

	sort.Slice(saves, func(i, j int) bool {
		if saves[i].LastMod.Equal(saves[j].LastMod) {
			return saves[i].Name < saves[j].Name
		}
		return saves[i].LastMod.After(saves[j].LastMod)
	})

	return saves, nil
}

func ListSavesWithLatestInDir(savesDir string) ([]Save, error) {
	saves, err := ListSavesInDir(savesDir)
	if err != nil || len(saves) == 0 {
		return saves, err
	}

	latestSave := saves[0]
	latestSave.Name = fmt.Sprintf("Load Latest (%s)", latestSave.Name)
	return append([]Save{latestSave}, saves...), nil
}

func FindSave(name string) (*Save, error) {
	config := bootstrap.GetConfig()
	savesDir := config.FactorioSavesDir
	if manager := GetServerManager(); manager != nil {
		savesDir = manager.DefaultServer().savesDir()
	}
	return FindSaveInDir(savesDir, name)
}

func FindSaveInDir(savesDir, name string) (*Save, error) {
	name, err := ValidateSaveName(name)
	if err != nil {
		return nil, err
	}

	saves, err := ListSavesInDir(savesDir)
	if err != nil {
		return nil, fmt.Errorf("error listing saves: %v", err)
	}

	for _, save := range saves {
		if save.Name == name {
			return &save, nil
		}
	}

	return nil, errors.New("save not found")
}

func (s *Save) Remove() error {
	if s.Name == "" {
		return errors.New("save name cannot be blank")
	}
	config := bootstrap.GetConfig()
	savesDir := config.FactorioSavesDir
	if manager := GetServerManager(); manager != nil {
		savesDir = manager.DefaultServer().savesDir()
	}
	return s.RemoveFromDir(savesDir)
}

func (s *Save) RemoveFromDir(savesDir string) error {
	savePath, err := SavePathInDir(savesDir, s.Name)
	if err != nil {
		return err
	}
	return os.Remove(savePath)
}

// Create savefiles for Factorio
func CreateSave(filePath string) (string, error) {
	config := bootstrap.GetConfig()
	binary := config.FactorioBinary
	if manager := GetServerManager(); manager != nil {
		binary = manager.DefaultServer().factorioBinary()
	}
	return CreateSaveWithBinary(filePath, binary)
}

func CreateSaveWithBinary(filePath, binary string) (string, error) {
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		log.Printf("Error in creating Factorio save: %s", err)
		return "", err
	}

	args := []string{"--create", filePath}
	cmdOutput, err := exec.Command(binary, args...).Output()
	if err != nil {
		log.Printf("Error in creating Factorio save: %s", err)
		log.Println(string(cmdOutput))
		return "", err
	}

	result := string(cmdOutput)

	return result, nil
}

func GetLatestSave() (save Save, err error) {
	config := bootstrap.GetConfig()
	savesDir := config.FactorioSavesDir
	if manager := GetServerManager(); manager != nil {
		savesDir = manager.DefaultServer().savesDir()
	}
	return GetLatestSaveInDir(savesDir)
}

func GetLatestSaveInDir(savesDir string) (save Save, err error) {
	saves, err := ListSavesInDir(savesDir)
	if err != nil || len(saves) == 0 {
		return save, err
	}
	save = saves[0]

	return
}

func ValidateSaveName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("save name cannot be blank")
	}
	if strings.HasPrefix(name, "Load Latest") {
		return "", errors.New("load latest is not a save file name")
	}
	if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", fmt.Errorf("invalid save name %q", name)
	}
	if !strings.EqualFold(filepath.Ext(name), ".zip") {
		return "", fmt.Errorf("save name must end with .zip")
	}
	// normalize extension to lowercase so Factorio binary finds the file
	ext := filepath.Ext(name)
	if ext != ".zip" {
		name = strings.TrimSuffix(name, ext) + ".zip"
	}
	if strings.TrimSuffix(name, ".zip") == "" {
		return "", errors.New("save name cannot be blank")
	}
	return name, nil
}

func SavePathInDir(savesDir, name string) (string, error) {
	name, err := ValidateSaveName(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(savesDir, name), nil
}

func SavePathForCreateInDir(savesDir, name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", errors.New("save name cannot be blank")
	}
	if filepath.Ext(name) == "" {
		name += ".zip"
	}
	name, err := ValidateSaveName(name)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(savesDir, name), name, nil
}
