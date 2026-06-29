package factorio

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

type LoginErrorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}
type LoginSuccessResponse struct {
	UserKey []string `json:""`
}

func DeleteAllMods() error {
	config := bootstrap.GetConfig()
	modsDir := config.FactorioModsDir
	if manager := GetServerManager(); manager != nil {
		modsDir = manager.DefaultServer().modsDir()
	}
	err := clearModsDir(modsDir)
	if err != nil {
		log.Printf("Error deleting all mods: %s", err)
		return err
	}
	return nil
}

func ClearModsDir(dir string) error {
	return clearModsDir(dir)
}

func ModStartUp() {
	config := bootstrap.GetConfig()
	factorioDir := config.FactorioDir
	modsDir := config.FactorioModsDir
	modPackDir := config.FactorioModPackDir
	if manager := GetServerManager(); manager != nil {
		server := manager.DefaultServer()
		factorioDir = server.Paths.Root
		modsDir = server.modsDir()
		modPackDir = server.Paths.ModPackDir
	}
	//get main-folder info
	factorioDirInfo, err := os.Stat(factorioDir)
	if err != nil {
		log.Printf("error getting stats from FactorioDir %s with error %s", factorioDir, err)
		return
	}
	factorioDirPerm := factorioDirInfo.Mode().Perm()

	//create mods dir
	if _, err = os.Stat(modsDir); os.IsNotExist(err) {
		log.Println("no mods dir found ... creating one ...")
		os.Mkdir(modsDir, factorioDirPerm)
	}

	//crate mod_pack dir
	if _, err = os.Stat(modPackDir); os.IsNotExist(err) {
		log.Println("no ModPackDir found ... creating one ...")
		_ = os.Mkdir(modPackDir, factorioDirPerm)
	}

	oldModpackDir := filepath.Join(factorioDir, "modpacks")
	if _, err := os.Stat(filepath.Join(oldModpackDir)); !os.IsNotExist(err) {
		log.Printf("found old modpack files, rebuild into new system...")

		err = filepath.Walk(oldModpackDir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(info.Name()) != ".zip" {
				log.Printf("file is not a zip or a directory -> skip")
				return nil
			}

			filename := info.Name()
			n := strings.LastIndexByte(info.Name(), '.')
			modPackName := filename[:n]

			log.Printf("loading modPack %s into new system ...", modPackName)

			modPackDir := filepath.Join(modPackDir, modPackName)

			if _, err := os.Stat(modPackDir); !os.IsNotExist(err) {
				log.Printf("modPack already exists")
				return errors.New("modPack already exists")
			}

			err = os.Mkdir(modPackDir, factorioDirPerm)
			if err != nil {
				log.Printf("error creating newModPackDir: %s", err)
				return err
			}

			//create mod-info.json
			modSimpleList := ModSimpleList{
				Destination: modPackDir,
				Mods: []ModSimple{
					ModSimple{
						Name:    "base",
						Enabled: true,
					},
				},
			}
			newJson, _ := json.Marshal(modSimpleList)

			err = ioutil.WriteFile(filepath.Join(modSimpleList.Destination, "mod-list.json"), newJson, 0664)
			if err != nil {
				log.Printf("error when writing new mod-list: %s", err)
				return err
			}

			modPackFile, err := zip.OpenReader(path)
			if err != nil {
				return err
			}
			defer modPackFile.Close()

			mods, err := NewMods(modPackDir)
			if err != nil {
				log.Printf("error reading mods: %s", err)
				return err
			}

			for _, modFile := range modPackFile.File {
				modFileRc, err := modFile.Open()
				if err != nil {
					log.Printf("error opening mod_file: %s", err)
					return err
				}
				defer modFileRc.Close()

				modFileBuffer, err := ioutil.ReadAll(modFileRc)
				if err != nil {
					log.Printf("error reading mod_file_rc: %s", err)
					return err
				}

				err = modFileRc.Close()
				if err != nil {
					log.Printf("error closing mod_file_rc: %s", err)
					return err
				}

				modFileByteReader := bytes.NewReader(modFileBuffer)
				modFileZipReader, err := zip.NewReader(modFileByteReader, int64(len(modFileBuffer)))
				if err != nil {
					log.Printf("error creating Reader on byte_array: %s", err)
					return err
				}

				var modInfo ModInfo
				err = modInfo.getModInfo(modFileZipReader)
				if err != nil {
					log.Printf("error loading the ModInfo: %s", err)
					return err
				}

				err = mods.createMod(modInfo.Name, modFile.Name, bytes.NewReader(modFileBuffer))
				if err != nil {
					log.Printf("error on creating mod: %s", err)
					return err
				}
			}

			log.Printf("loading modPack %s successful", modPackName)

			return nil
		})

		if err != nil {
			log.Printf("error on loading old modpacks into the new system: %s\n please check if empty modPacks are creating and delete them", err)
		} else {
			log.Printf("all modPacks are loaded into the new system successfully")
			log.Printf("deleting old modPackDir")
			os.RemoveAll(oldModpackDir)
		}
	}
}

// clearModsDir удаляет содержимое директории, но не саму директорию.
// Используется вместо os.RemoveAll когда директория примонтирована как Docker volume.
func clearModsDir(dir string) error {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("clearModsDir ReadDir: %v", err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("clearModsDir remove %s: %v", entry.Name(), err)
		}
	}
	return nil
}
