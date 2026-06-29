package factorio

import (
	"io"
	"sync"
)

type InstallStatus struct {
	Installing bool   `json:"installing"`
	Version    string `json:"version"`
	Progress   int    `json:"progress"`
	Error      string `json:"error,omitempty"`
}

var installMu sync.Mutex
var currentInstall InstallStatus

func GetInstallStatus() InstallStatus {
	installMu.Lock()
	defer installMu.Unlock()
	return currentInstall
}

func SetInstallStatus(s InstallStatus) {
	installMu.Lock()
	defer installMu.Unlock()
	currentInstall = s
}

type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Current    int64
	OnProgress func(percent int)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Total > 0 && pr.OnProgress != nil {
		pct := int(float64(pr.Current) / float64(pr.Total) * 100)
		pr.OnProgress(pct)
	}
	return n, err
}
