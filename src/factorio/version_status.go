package factorio

import "sync"

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
