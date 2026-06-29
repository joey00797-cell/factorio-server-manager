package api

import (
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func serverFromRequest(r *http.Request) (*factorio.Server, bool) {
	id := mux.Vars(r)["serverID"]
	if id == "" {
		id = factorio.DefaultServerID
	}
	manager := factorio.GetServerManager()
	if manager == nil {
		if id == factorio.DefaultServerID {
			return factorio.GetFactorioServer(), true
		}
		return nil, false
	}
	return manager.GetServer(id)
}

func serverSavesDir(server *factorio.Server) string {
	if server != nil && server.Paths.SavesDir != "" {
		return server.Paths.SavesDir
	}
	return bootstrap.GetConfig().FactorioSavesDir
}

func serverModsDir(server *factorio.Server) string {
	if server != nil && server.Paths.ModsDir != "" {
		return server.Paths.ModsDir
	}
	return bootstrap.GetConfig().FactorioModsDir
}

func serverConfigFile(server *factorio.Server) string {
	if server != nil && server.Paths.ConfigFile != "" {
		return server.Paths.ConfigFile
	}
	return bootstrap.GetConfig().FactorioConfigFile
}

func serverSettingsFile(server *factorio.Server) string {
	if server != nil && server.Paths.SettingsFile != "" {
		return server.Paths.SettingsFile
	}
	return bootstrap.GetConfig().SettingsFile
}

func serverAdminFile(server *factorio.Server) string {
	if server != nil && server.Paths.AdminFile != "" {
		return server.Paths.AdminFile
	}
	return bootstrap.GetConfig().FactorioAdminFile
}

func serverLogFile(server *factorio.Server) string {
	if server != nil && server.Paths.FactorioLog != "" {
		return server.Paths.FactorioLog
	}
	return bootstrap.GetConfig().FactorioLog
}

func serverConsoleLogFile(server *factorio.Server) string {
	if server != nil && server.Paths.ConsoleLogFile != "" {
		return server.Paths.ConsoleLogFile
	}
	return bootstrap.GetConfig().ConsoleLogFile
}

func serverLogFiles(server *factorio.Server) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, path := range []string{serverConsoleLogFile(server), serverLogFile(server)} {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		result = append(result, path)
	}
	return result
}

func serverBinary(server *factorio.Server) string {
	if server != nil && server.Paths.FactorioBinary != "" {
		return server.Paths.FactorioBinary
	}
	return bootstrap.GetConfig().FactorioBinary
}
