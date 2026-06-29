package factorio

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/rcon"
)

type Server struct {
	Cmd            *exec.Cmd              `json:"-"`
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	VersionLabel   string                 `json:"version"`
	Savefile       string                 `json:"savefile"`
	Latency        int                    `json:"latency"`
	BindIP         string                 `json:"bindip"`
	Port           int                    `json:"port"`
	RconPort       int                    `json:"rcon_port"`
	RconPass       string                 `json:"-"`
	Autostart      bool                   `json:"autostart"`
	PendingRestart bool                   `json:"pending_restart"`
	Paths          InstancePaths          `json:"paths"`
	Running        bool                   `json:"running"`
	Version        Version                `json:"fac_version"`
	BaseModVersion string                 `json:"base_mod_version"`
	StdOut         io.ReadCloser          `json:"-"`
	StdErr         io.ReadCloser          `json:"-"`
	StdIn          io.WriteCloser         `json:"-"`
	Settings       map[string]interface{} `json:"-"`
	Rcon           *rcon.RemoteConsole    `json:"-"`
	LogChan        chan []string          `json:"-"`
}

var instantiated Server
var once sync.Once

func (server *Server) SetRunning(newState bool) {
	if server.Running != newState {
		log.Println("new state, will also send to correct room")
		server.Running = newState
		response, _ := json.Marshal(server)
		websocket.WebsocketHub.GetRoom("server_status").Send(string(response))
		websocket.WebsocketHub.GetRoom("servers").Send(string(response))
		if server.ID != "" {
			websocket.WebsocketHub.GetRoom("servers:" + server.ID + ":status").Send(string(response))
		}
	}
}

func (server *Server) GetRunning() bool {
	return server.Running
}

func (server *Server) autostart() {
	var err error
	if server.BindIP == "" {
		server.BindIP = "0.0.0.0"

	}
	if server.Port == 0 {
		server.Port = 34197
	}
	server.Savefile = "Load Latest"

	err = server.Run()

	if err != nil {
		log.Printf("Error starting Factorio server: %+v", err)
		return
	}

}

func SetFactorioServer(server Server) {
	instantiated = server
}

func NewFactorioServer() (err error) {
	server := Server{}
	server.Settings = make(map[string]interface{})
	config := bootstrap.GetConfig()
	if err = os.MkdirAll(config.FactorioConfigDir, 0755); err != nil {
		log.Printf("failed to create config directory: %v", err)
		return
	}

	settingsPath := config.SettingsFile
	var settings *os.File

	log.Printf("DEBUG: checking settings file: %s", settingsPath)
	if _, err = os.Stat(settingsPath); os.IsNotExist(err) {
		log.Printf("DEBUG: settings file NOT FOUND, loading default config")

		defaultContent := getDefaultServerSettings()

		os.MkdirAll(filepath.Dir(settingsPath), 0755)
		if err = os.WriteFile(settingsPath, []byte(defaultContent), 0644); err != nil {
			log.Printf("failed to write default server settings: %v", err)
			return
		}
		log.Printf("Default server-settings.json written to %s", settingsPath)

		settings, err = os.Open(settingsPath)
		if err != nil {
			log.Printf("failed to open server settings file: %v", err)
			return
		}
		defer settings.Close()
	} else {
		log.Printf("DEBUG: settings file EXISTS, opening normally")
		// otherwise, open file normally
		settings, err = os.Open(settingsPath)
		if err != nil {
			log.Printf("failed to open server settings file: %v", err)
			return
		}
		defer settings.Close()
	}

	// before reading reset offset
	if _, err = settings.Seek(0, 0); err != nil {
		log.Printf("error while seeking in settings file: %v", err)
		return
	}

	if err = json.NewDecoder(settings).Decode(&server.Settings); err != nil {
		log.Printf("error reading %s: %v", settingsPath, err)
		return
	}
	// Защита от null в JSON файле
	if server.Settings == nil {
		log.Printf("server-settings.json содержит null, инициализируем пустыми настройками")
		server.Settings = make(map[string]interface{})
	}

	log.Printf("Loaded Factorio settings from %s\n", settingsPath)

	out := []byte{}
	//Load factorio version
	if config.GlibcCustom == "true" {
		out, err = exec.Command(config.GlibcLocation, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--version").Output()
	} else {
		out, err = exec.Command(config.FactorioBinary, "--version").Output()
	}

	if err != nil {
		log.Printf("error on loading factorio version: %s", err)
		return
	}

	reg := regexp.MustCompile("Version.*?((\\d+\\.)?(\\d+\\.)?(\\*|\\d+)+)")
	found := reg.FindStringSubmatch(string(out))
	err = server.Version.UnmarshalText([]byte(found[1]))
	if err != nil {
		log.Printf("could not parse version: %v", err)
		return
	}

	//Load baseMod version
	baseModInfoFile := filepath.Join(config.FactorioBaseModDir, "info.json")
	bmifBa, err := ioutil.ReadFile(baseModInfoFile)
	if err != nil {
		log.Printf("couldn't open baseMods info.json: %s", err)
		return
	}
	var modInfo ModInfo
	err = json.Unmarshal(bmifBa, &modInfo)
	if err != nil {
		log.Printf("error unmarshalling baseMods info.json to a modInfo: %s", err)
		return
	}

	server.BaseModVersion = modInfo.Version

	// load admins from additional file
	if (server.Version.Greater(Version{0, 17, 0})) {
		if _, err = os.Stat(config.FactorioAdminFile); os.IsNotExist(err) {
			//save empty admins-file
			err = ioutil.WriteFile(config.FactorioAdminFile, []byte("[]"), 0664)
			server.Settings["admins"] = make([]string, 0)
		} else {
			var data []byte
			data, err = ioutil.ReadFile(config.FactorioAdminFile)
			if err != nil {
				log.Printf("Error loading FactorioAdminFile: %s", err)
				return
			}

			var jsonData interface{}
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				log.Printf("Error unmarshalling FactorioAdminFile: %s", err)
				return
			}

			server.Settings["admins"] = jsonData
		}
	}

	SetFactorioServer(server)

	// autostart factorio is configured to do so
	if config.Autostart == "true" {
		go instantiated.autostart()
	}

	return
}

func GetFactorioServer() (f *Server) {
	return &instantiated
}

func (server *Server) Run() error {
	var err error
	if manager := GetServerManager(); manager != nil {
		if err := manager.EnsureServerVersion(server); err != nil {
			return err
		}
	}
	config := bootstrap.GetConfig()
	settingsFile := server.settingsFile()
	binary := server.factorioBinary()
	adminFile := server.adminFile()
	savesDir := server.savesDir()
	consoleLogFile := server.consoleLogFile()
	configFile := server.configFile()
	modsDir := server.modsDir()
	rconPort := server.rconPort()
	rconPass := server.rconPass()

	log.Printf("DEBUG Run(): starting, Savefile=%s BindIP=%s Port=%d", server.Savefile, server.BindIP, server.Port)
	log.Printf("DEBUG Run(): Settings keys count=%d", len(server.Settings))
	log.Printf("DEBUG Run(): SettingsFile=%s", settingsFile)
	log.Printf("DEBUG Run(): FactorioBinary=%s", binary)
	data, err := json.MarshalIndent(server.Settings, "", "  ")
	if err != nil {
		log.Println("Failed to marshal FactorioServerSettings: ", err)
	} else if len(server.Settings) < 5 {
		log.Printf("WARNING: server.Settings has only %d keys, skipping write to prevent corruption", len(server.Settings))
	} else {
		log.Printf("DEBUG Run(): writing %d settings keys to %s", len(server.Settings), settingsFile)
		ioutil.WriteFile(settingsFile, data, 0644)
	}

	saves, err := ListSavesInDir(savesDir)
	if err != nil {
		log.Println("Failed to get saves list: ", err)
	}

	if len(saves) == 0 {
		return errors.New("No savefile exists on the server")
	}

	args := []string{}

	//The factorio server refenences its executable-path, since we execute the ld.so file and pass the factorio binary as a parameter
	//the game would use the path to the ld.so file as it's executable path and crash, to prevent this the parameter "--executable-path" is added
	if config.GlibcCustom == "true" {
		log.Println("Custom glibc selected, glibc.so location:", config.GlibcLocation, " lib location:", config.GlibcLibLoc)
		args = append(args, "--library-path", config.GlibcLibLoc, binary, "--executable-path", binary)
	}

	args = append(args,
		"--config", configFile,
		"--mod-directory", modsDir,
		"--bind", server.BindIP,
		"--port", strconv.Itoa(server.Port),
		"--server-settings", settingsFile,
		"--rcon-port", strconv.Itoa(rconPort),
		"--rcon-password", rconPass)

	if (server.Version.Greater(Version{0, 17, 0})) {
		args = append(args, "--server-adminlist", adminFile)
	}

	if strings.HasPrefix(server.Savefile, "Load Latest") {
		args = append(args, "--start-server-load-latest")
	} else {
		args = append(args, "--start-server", filepath.Join(savesDir, server.Savefile))
	}

	// Write chat log to a different file if requested (if not it will be mixed-in with the default logfile)
	if config.ChatLogFile != "" {
		args = append(args, "--console-log", config.ChatLogFile)
	} else if consoleLogFile != "" {
		args = append(args, "--console-log", consoleLogFile)
	}

	if config.GlibcCustom == "true" {
		log.Println("Starting server with command: ", config.GlibcLocation, redactedServerArgs(args))
		server.Cmd = exec.Command(config.GlibcLocation, args...)
	} else {
		log.Println("Starting server with command: ", binary, redactedServerArgs(args))
		server.Cmd = exec.Command(binary, args...)
	}

	server.StdOut, err = server.Cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error opening stdout pipe: %s", err)
		return err
	}

	server.StdIn, err = server.Cmd.StdinPipe()
	if err != nil {
		log.Printf("Error opening stdin pipe: %s", err)
		return err
	}

	server.StdErr, err = server.Cmd.StderrPipe()
	if err != nil {
		log.Printf("Error opening stderr pipe: %s", err)
		return err
	}

	go server.parseRunningCommand(server.StdOut)
	go server.parseRunningCommand(server.StdErr)

	// Ждём завершения синка модов если он идёт
	if IsModsSyncingForServer(server.ID) {
		log.Println("Waiting for mod sync to complete before starting server...")
		for IsModsSyncingForServer(server.ID) {
			time.Sleep(1 * time.Second)
		}
		log.Println("Mod sync complete, starting server")
	}

	err = server.Cmd.Start()
	if err != nil {
		log.Printf("Factorio process failed to start: %s", err)
		return err
	}
	server.SetRunning(true)

	err = server.Cmd.Wait()
	log.Printf("Factorio process is closed")
	server.SetRunning(false)
	if err != nil {
		log.Printf("Factorio process exited with error: %s", err)
		return err
	}

	return nil
}

func (server *Server) parseRunningCommand(std io.ReadCloser) (err error) {
	stdScanner := bufio.NewScanner(std)
	for stdScanner.Scan() {
		text := stdScanner.Text()

		if err := server.writeLog(text); err != nil {
			log.Printf("Error: %s", err)
		}

		// send the reported line per websocket
		if server.ID == "" || server.ID == DefaultServerID {
			go websocket.WebsocketHub.GetRoom("gamelog").Send(text)
		}
		if server.ID != "" {
			go websocket.WebsocketHub.GetRoom("servers:" + server.ID + ":gamelog").Send(text)
		}

		line := strings.Fields(text)
		// Ensure logline slice is in bounds
		if len(line) > 1 {
			// Check if Factorio Server reports any errors if so handle it
			if line[1] == "Error" {
				log.Printf("Factorio server %s reported error: %s", server.ID, text)
				err := server.checkLogError(line)
				if err != nil {
					log.Printf("Error checking Factorio Server Error: %s", err)
				}
			}
			// If rcon port opens indicated in log connect to rcon
			rconLog := "Starting RCON interface at IP"
			// check if slice index is greater than 2 to prevent panic
			if len(line) > 2 {
				// log line for opened rcon connection
				if strings.Contains(text, rconLog) {
					log.Printf("Rcon running on Factorio Server")
					err = server.connectRC()
					if err != nil {
						log.Printf("Error: %s", err)
					}
				}

				server.checkProcessHealth(text)
			}
		}
	}
	if err := stdScanner.Err(); err != nil {
		log.Printf("Error reading std buffer: %s", err)
		return err
	}
	return nil
}

func (server *Server) writeLog(logline string) error {
	logfileName := server.consoleLogFile()
	file, err := os.OpenFile(logfileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Cannot open logfile %s for appending Factorio Server output: %s", logfileName, err)
		return err
	}
	defer file.Close()

	logline = logline + "\n"

	if _, err = file.WriteString(logline); err != nil {
		log.Printf("Error appending to %s: %s", logfileName, err)
		return err
	}

	return nil
}

func (server *Server) checkLogError(logline []string) error {
	// TODO Handle errors generated by running Factorio Server
	return nil
}

func redactedServerArgs(args []string) []string {
	redacted := append([]string(nil), args...)
	for i := range redacted {
		if redacted[i] == "--rcon-password" && i+1 < len(redacted) {
			redacted[i+1] = "[REDACTED]"
		}
	}
	return redacted
}

func init() {
	websocket.WebsocketHub.RegisterControlHandler <- serverWebsocketControl
}

// react to websocket control messages and run the command if it is requested
func serverWebsocketControl(controls websocket.WsControls) {
	log.Println(controls)
	if controls.Type == "command" {
		command := controls.Value
		serverID := DefaultServerID
		var payload struct {
			ServerID string `json:"serverId"`
			Command  string `json:"command"`
		}
		if strings.HasPrefix(strings.TrimSpace(controls.Value), "{") {
			if err := json.Unmarshal([]byte(controls.Value), &payload); err == nil {
				if payload.ServerID != "" {
					serverID = payload.ServerID
				}
				if payload.Command != "" {
					command = payload.Command
				}
			}
		}
		server := GetFactorioServer()
		if manager := GetServerManager(); manager != nil {
			if targeted, ok := manager.GetServer(serverID); ok {
				server = targeted
			}
		}
		if server.GetRunning() {
			log.Printf("Received command: %v", command)

			reqId, err := server.Rcon.Write(command)
			if err != nil {
				log.Printf("Error sending rcon command: %s", err)
				return
			}

			log.Printf("Command send to Factorio: %s, with rcon request id: %v", command, reqId)
		}
	}
}

// RefreshVersion перечитывает версию Factorio бинарника
func (s *Server) RefreshVersion() error {
	out, err := exec.Command(s.factorioBinary(), "--version").Output()
	if err != nil {
		return err
	}
	// Парсим версию из вывода
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Version:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if err := s.Version.UnmarshalText([]byte(parts[1])); err == nil {
					log.Printf("Factorio version updated: %s", s.Version.String())
					break
				}
			}
		}
	}
	return nil
}

func (server *Server) factorioBinary() string {
	if server.Paths.FactorioBinary != "" {
		return server.Paths.FactorioBinary
	}
	return bootstrap.GetConfig().FactorioBinary
}

func (server *Server) settingsFile() string {
	if server.Paths.SettingsFile != "" {
		return server.Paths.SettingsFile
	}
	return bootstrap.GetConfig().SettingsFile
}

func (server *Server) adminFile() string {
	if server.Paths.AdminFile != "" {
		return server.Paths.AdminFile
	}
	return bootstrap.GetConfig().FactorioAdminFile
}

func (server *Server) savesDir() string {
	if server.Paths.SavesDir != "" {
		return server.Paths.SavesDir
	}
	return bootstrap.GetConfig().FactorioSavesDir
}

func (server *Server) modsDir() string {
	if server.Paths.ModsDir != "" {
		return server.Paths.ModsDir
	}
	return bootstrap.GetConfig().FactorioModsDir
}

func (server *Server) configFile() string {
	if server.Paths.ConfigFile != "" {
		return server.Paths.ConfigFile
	}
	return bootstrap.GetConfig().FactorioConfigFile
}

func (server *Server) consoleLogFile() string {
	if server.Paths.ConsoleLogFile != "" {
		return server.Paths.ConsoleLogFile
	}
	return bootstrap.GetConfig().ConsoleLogFile
}

func (server *Server) factorioLogFile() string {
	if server.Paths.FactorioLog != "" {
		return server.Paths.FactorioLog
	}
	return bootstrap.GetConfig().FactorioLog
}

func (server *Server) rconPort() int {
	if server.RconPort != 0 {
		return server.RconPort
	}
	return bootstrap.GetConfig().FactorioRconPort
}

func (server *Server) rconPass() string {
	if server.RconPass != "" {
		return server.RconPass
	}
	return bootstrap.GetConfig().FactorioRconPass
}

func (server *Server) logRoomName() string {
	if server.ID == "" || server.ID == DefaultServerID {
		return "gamelog"
	}
	return "servers:" + server.ID + ":gamelog"
}
