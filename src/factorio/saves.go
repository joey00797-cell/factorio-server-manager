package factorio

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	err = filepath.Walk(savesDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || (info.IsDir() && info.Name() == "saves") {
			return nil
		}
		saves = append(saves, Save{
			info.Name(),
			info.ModTime(),
			info.Size(),
		})
		return nil
	})
	return
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
	if s.Name == "" {
		return errors.New("save name cannot be blank")
	}
	return os.Remove(filepath.Join(savesDir, s.Name))
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
	err = filepath.Walk(savesDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || (info.IsDir() && info.Name() == "saves") {
			return nil
		}

		if save.LastMod.Before(info.ModTime()) {
			save = Save{
				Name:    info.Name(),
				LastMod: info.ModTime(),
				Size:    info.Size(),
			}
		}
		return nil
	})

	return
}
