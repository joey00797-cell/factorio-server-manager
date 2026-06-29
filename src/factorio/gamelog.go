package factorio

import (
	"bytes"
	"os"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

const maxTailLogLines = 200

func TailLog() ([]string, error) {
	config := bootstrap.GetConfig()
	logFile := config.ConsoleLogFile
	if manager := GetServerManager(); manager != nil {
		logFile = manager.DefaultServer().consoleLogFile()
	}
	return TailLogFile(logFile)
}

func TailLogFile(logFile string) ([]string, error) {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return []string{}, err
	}
	data = bytes.TrimRight(data, "\r\n")
	if len(data) == 0 {
		return []string{}, nil
	}

	lines := bytes.Split(data, []byte("\n"))
	start := 0
	if len(lines) > maxTailLogLines {
		start = len(lines) - maxTailLogLines
	}
	result := make([]string, 0, len(lines)-start)
	for _, line := range lines[start:] {
		result = append(result, strings.TrimRight(string(line), "\r"))
	}
	return result, nil
}
