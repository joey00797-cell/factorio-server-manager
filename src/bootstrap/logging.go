package bootstrap

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FSMLogFile returns the persistent log file used for Factorio Server Manager
// process logs. It intentionally does not point at any per-server Factorio log.
func FSMLogFile() string {
	config := GetConfig()
	if strings.TrimSpace(config.LogFile) != "" {
		return config.LogFile
	}
	root := config.ServersRoot
	if strings.TrimSpace(root) == "" {
		root = "/opt/factorio-server"
	}
	return filepath.Join(root, "logs", "fsm.log")
}

func ConfigureFSMLogging() (*os.File, error) {
	logFile := FSMLogFile()
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	log.SetOutput(io.MultiWriter(os.Stderr, file))
	return file, nil
}
