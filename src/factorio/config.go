package factorio

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-ini/ini"
)

var saveConfigMu sync.Mutex

// Loads config.ini file from the factorio bootstrap directory
func LoadConfig(filename string) (map[string]map[string]string, error) {
	log.Printf("Loading config file: %s", filename)
	cfg, err := ini.Load(filename)
	if err != nil {
		log.Printf("Error loading config.ini file: %s", err)
		return nil, err
	}

	result := map[string]map[string]string{}

	sections := cfg.Sections()
	sectionNames := cfg.SectionStrings()
	log.Printf("Appending sections %s to JSON response", sectionNames)
	for _, s := range sections {
		sectionName := s.Name()
		if sectionName == "DEFAULT" {
			continue
		}
		result[sectionName] = map[string]string{}
		result[sectionName] = s.KeysHash()
	}
	log.Printf("Encoding config.ini to JSON")

	return result, nil
}

func SaveConfig(filename string, data map[string]map[string]string) error {
	log.Printf("Saving config file: %s", filename)
	cfg := ini.Empty()
	for sectionName, values := range data {
		section, err := cfg.NewSection(sectionName)
		if err != nil {
			return err
		}
		for key, value := range values {
			if _, err := section.NewKey(key, value); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	saveConfigMu.Lock()
	defer saveConfigMu.Unlock()
	previousPrettyFormat := ini.PrettyFormat
	previousPrettyEqual := ini.PrettyEqual
	previousFormatLeft := ini.DefaultFormatLeft
	previousFormatRight := ini.DefaultFormatRight
	ini.PrettyFormat = false
	ini.PrettyEqual = false
	ini.DefaultFormatLeft = ""
	ini.DefaultFormatRight = ""
	defer func() {
		ini.PrettyFormat = previousPrettyFormat
		ini.PrettyEqual = previousPrettyEqual
		ini.DefaultFormatLeft = previousFormatLeft
		ini.DefaultFormatRight = previousFormatRight
	}()

	return cfg.SaveTo(filename)
}
