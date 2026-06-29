package factorio

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

const DefaultServerID = "1"

type InstancePaths struct {
	Root           string `json:"root"`
	VersionDir     string `json:"version_dir"`
	FactorioBinary string `json:"factorio_binary"`
	BaseModDir     string `json:"base_mod_dir"`
	SavesDir       string `json:"saves_dir"`
	ModsDir        string `json:"mods_dir"`
	ModPackDir     string `json:"mod_pack_dir"`
	ConfigDir      string `json:"config_dir"`
	ConfigFile     string `json:"config_file"`
	SettingsFile   string `json:"settings_file"`
	AdminFile      string `json:"admin_file"`
	LogsDir        string `json:"logs_dir"`
	ConsoleLogFile string `json:"console_log_file"`
	FactorioLog    string `json:"factorio_log"`
	ModSettingsDat string `json:"mod_settings_dat"`
}

type ServerRecord struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Version        string        `json:"version"`
	BindIP         string        `json:"bind_ip"`
	Port           int           `json:"port"`
	RconPort       int           `json:"rcon_port"`
	RconPass       string        `json:"rcon_pass"`
	Autostart      bool          `json:"autostart"`
	LastSave       string        `json:"last_save"`
	PendingRestart bool          `json:"pending_restart"`
	Paths          InstancePaths `json:"paths"`
}

type ServerCatalog struct {
	Version int            `json:"version"`
	NextID  int            `json:"next_id"`
	Servers []ServerRecord `json:"servers"`
}

type ServerManager struct {
	mu          sync.RWMutex
	root        string
	catalogFile string
	downloadDir string
	versionDir  string
	instanceDir string
	installMu   sync.Mutex
	catalog     ServerCatalog
	servers     map[string]*Server
}

var serverManager *ServerManager

func InitServerManager() (*ServerManager, error) {
	config := bootstrap.GetConfig()
	manager := &ServerManager{
		root:        config.ServersRoot,
		catalogFile: filepath.Join(config.ServersRoot, "servers.json"),
		downloadDir: filepath.Join(config.ServersRoot, "downloads"),
		versionDir:  filepath.Join(config.ServersRoot, "versions"),
		instanceDir: filepath.Join(config.ServersRoot, "instances"),
		servers:     make(map[string]*Server),
	}

	if err := manager.ensureLayout(); err != nil {
		return nil, err
	}

	if _, err := os.Stat(manager.catalogFile); os.IsNotExist(err) {
		if err := manager.migrateLegacy(config); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if err := manager.loadCatalog(); err != nil {
		return nil, err
	}
	if err := manager.rebuildServers(); err != nil {
		return nil, err
	}

	serverManager = manager
	SetFactorioServer(*manager.DefaultServer())
	return manager, nil
}

func GetServerManager() *ServerManager {
	return serverManager
}

func (m *ServerManager) ensureLayout() error {
	for _, dir := range []string{m.root, m.downloadDir, m.versionDir, m.instanceDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (m *ServerManager) loadCatalog() error {
	raw, err := os.ReadFile(m.catalogFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &m.catalog); err != nil {
		return err
	}
	if m.catalog.Version == 0 {
		m.catalog.Version = 1
	}
	if m.catalog.NextID <= 1 {
		m.catalog.NextID = 2
	}
	return nil
}

func (m *ServerManager) saveCatalogLocked() error {
	tmp := m.catalogFile + ".tmp"
	raw, err := json.MarshalIndent(m.catalog, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.catalogFile); err != nil {
		_ = os.Remove(m.catalogFile)
		return os.Rename(tmp, m.catalogFile)
	}
	return nil
}

func (m *ServerManager) SaveCatalog() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCatalogLocked()
}

func (m *ServerManager) migrateLegacy(config bootstrap.Config) error {
	log.Printf("No managed server catalog found; migrating legacy server into ID %s", DefaultServerID)

	version := detectFactorioVersion(config.FactorioBinary, config.GlibcCustom, config.GlibcLocation, config.GlibcLibLoc)
	if version == "" {
		version = "uninstalled"
	}

	versionDir := filepath.Join(m.versionDir, version)
	instanceRoot := filepath.Join(m.instanceDir, DefaultServerID)
	paths := buildInstancePaths(instanceRoot, versionDir)
	if err := createInstanceDirs(paths); err != nil {
		return err
	}

	if version != "uninstalled" {
		if err := migrateLegacyInstall(config.FactorioDir, versionDir); err != nil {
			log.Printf("Legacy install migration warning: %v", err)
		}
	}

	moves := map[string]string{
		config.FactorioSavesDir:   paths.SavesDir,
		config.FactorioModsDir:    paths.ModsDir,
		config.FactorioConfigDir:  paths.ConfigDir,
		config.FactorioModPackDir: paths.ModPackDir,
	}
	for src, dst := range moves {
		if src == "" {
			continue
		}
		if err := movePathIfExists(src, dst); err != nil {
			log.Printf("Legacy data migration warning %s -> %s: %v", src, dst, err)
		}
	}

	logMoves := map[string]string{
		config.ConsoleLogFile: filepath.Join(paths.LogsDir, filepath.Base(paths.ConsoleLogFile)),
		config.FactorioLog:    filepath.Join(paths.LogsDir, filepath.Base(paths.FactorioLog)),
	}
	for src, dst := range logMoves {
		if err := movePathIfExists(src, dst); err != nil {
			log.Printf("Legacy log migration warning %s -> %s: %v", src, dst, err)
		}
	}

	if err := EnsureInstanceFiles(paths); err != nil {
		return err
	}

	port := 34197
	rconPort := config.FactorioRconPort
	if rconPort == 0 {
		rconPort = 40000
	}

	m.catalog = ServerCatalog{
		Version: 1,
		NextID:  2,
		Servers: []ServerRecord{
			{
				ID:        DefaultServerID,
				Name:      "Server 1",
				Version:   version,
				BindIP:    defaultString(config.FactorioIP, "0.0.0.0"),
				Port:      port,
				RconPort:  rconPort,
				RconPass:  config.FactorioRconPass,
				Autostart: config.Autostart == "true" || config.AutostartServer,
				Paths:     paths,
			},
		},
	}

	if err := m.saveCatalogLocked(); err != nil {
		return err
	}

	marker := filepath.Join(instanceRoot, ".legacy-migrated.json")
	markerRaw, _ := json.MarshalIndent(map[string]string{
		"server_id": DefaultServerID,
		"version":   version,
	}, "", "  ")
	_ = os.WriteFile(marker, markerRaw, 0644)
	return nil
}

func (m *ServerManager) rebuildServers() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := make(map[string]*Server)
	for _, record := range m.catalog.Servers {
		server := serverFromRecord(record)
		if err := server.loadMetadata(); err != nil {
			log.Printf("Server %s metadata warning: %v", record.ID, err)
		}
		next[record.ID] = server
	}
	m.servers = next
	return nil
}

func (m *ServerManager) ListServers() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.servers))
	for id := range m.servers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		ii, _ := strconv.Atoi(ids[i])
		jj, _ := strconv.Atoi(ids[j])
		if ii == 0 || jj == 0 {
			return ids[i] < ids[j]
		}
		return ii < jj
	})

	result := make([]*Server, 0, len(ids))
	for _, id := range ids {
		result = append(result, m.servers[id])
	}
	return result
}

func (m *ServerManager) GetServer(id string) (*Server, bool) {
	if id == "" {
		id = DefaultServerID
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	server, ok := m.servers[id]
	return server, ok
}

func (m *ServerManager) DefaultServer() *Server {
	server, ok := m.GetServer(DefaultServerID)
	if ok {
		return server
	}
	return &instantiated
}

func (m *ServerManager) StartAutostartServers() {
	for _, server := range m.ListServers() {
		if !server.Autostart {
			continue
		}
		go func(s *Server) {
			if s.BindIP == "" {
				s.BindIP = "0.0.0.0"
			}
			if s.Port == 0 {
				s.Port = m.nextAvailableGamePort()
			}
			if s.Savefile == "" {
				s.Savefile = "Load Latest"
			}
			if err := s.Run(); err != nil {
				log.Printf("Error autostarting Factorio server %s: %+v", s.ID, err)
			}
		}(server)
	}
}

func (m *ServerManager) CreateServer(name, version, bindIP string, port int, autostart bool) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("Server %d", m.catalog.NextID)
	}
	if strings.TrimSpace(version) == "" || version == "uninstalled" {
		version = "stable"
	}
	resolvedVersion, err := resolveFactorioVersion(version)
	if err != nil {
		return nil, err
	}
	if bindIP == "" {
		bindIP = "0.0.0.0"
	}
	if port == 0 {
		port = m.nextAvailableGamePortLocked()
	}

	id := strconv.Itoa(m.catalog.NextID)
	m.catalog.NextID++
	versionDir := filepath.Join(m.versionDir, resolvedVersion)
	instanceRoot := filepath.Join(m.instanceDir, id)
	paths := buildInstancePaths(instanceRoot, versionDir)
	if err := EnsureInstanceFiles(paths); err != nil {
		return nil, err
	}

	record := ServerRecord{
		ID:        id,
		Name:      name,
		Version:   resolvedVersion,
		BindIP:    bindIP,
		Port:      port,
		RconPort:  m.nextAvailableRconPortLocked(),
		RconPass:  bootstrap.GenerateRandomPassword(),
		Autostart: autostart,
		Paths:     paths,
	}
	m.catalog.Servers = append(m.catalog.Servers, record)
	if err := m.saveCatalogLocked(); err != nil {
		return nil, err
	}

	server := serverFromRecord(record)
	if err := server.loadMetadata(); err != nil {
		log.Printf("Server %s metadata warning: %v", record.ID, err)
	}
	m.servers[id] = server
	return server, nil
}

func (m *ServerManager) UpdateServer(server *Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.catalog.Servers {
		if m.catalog.Servers[i].ID == server.ID {
			m.catalog.Servers[i].Name = server.Name
			m.catalog.Servers[i].BindIP = server.BindIP
			m.catalog.Servers[i].Port = server.Port
			m.catalog.Servers[i].RconPort = server.RconPort
			m.catalog.Servers[i].RconPass = server.RconPass
			m.catalog.Servers[i].Autostart = server.Autostart
			m.catalog.Servers[i].LastSave = server.Savefile
			m.catalog.Servers[i].PendingRestart = server.PendingRestart
			m.catalog.Servers[i].Version = server.VersionLabel
			m.catalog.Servers[i].Paths = server.Paths
			return m.saveCatalogLocked()
		}
	}
	return fmt.Errorf("server %s not found", server.ID)
}

func (m *ServerManager) DeleteServer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.catalog.Servers {
		if m.catalog.Servers[i].ID == id {
			m.catalog.Servers = append(m.catalog.Servers[:i], m.catalog.Servers[i+1:]...)
			delete(m.servers, id)
			return m.saveCatalogLocked()
		}
	}
	return fmt.Errorf("server %s not found", id)
}

func (m *ServerManager) nextAvailableGamePortLocked() int {
	start, end := parsePortRange(bootstrap.GetConfig().GamePortRange)
	used := make(map[int]bool)
	for _, srv := range m.catalog.Servers {
		used[srv.Port] = true
	}
	for port := start; port <= end; port++ {
		if !used[port] {
			return port
		}
	}
	return end + 1
}

func (m *ServerManager) nextAvailableGamePort() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextAvailableGamePortLocked()
}

func (m *ServerManager) nextAvailableRconPortLocked() int {
	used := make(map[int]bool)
	for _, srv := range m.catalog.Servers {
		used[srv.RconPort] = true
	}
	for port := 40000; port <= 45000; port++ {
		if !used[port] {
			return port
		}
	}
	return 45001
}

func serverFromRecord(record ServerRecord) *Server {
	return &Server{
		ID:             record.ID,
		Name:           record.Name,
		VersionLabel:   record.Version,
		Savefile:       record.LastSave,
		BindIP:         defaultString(record.BindIP, "0.0.0.0"),
		Port:           record.Port,
		RconPort:       record.RconPort,
		RconPass:       record.RconPass,
		Autostart:      record.Autostart,
		PendingRestart: record.PendingRestart,
		Paths:          record.Paths,
		Settings:       make(map[string]interface{}),
	}
}

func (server *Server) loadMetadata() error {
	if server.Settings == nil {
		server.Settings = make(map[string]interface{})
	}
	if _, err := os.Stat(server.Paths.SettingsFile); os.IsNotExist(err) {
		if err := EnsureInstanceFiles(server.Paths); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(server.Paths.SettingsFile)
	if err == nil {
		_ = json.Unmarshal(raw, &server.Settings)
	}
	if server.Settings == nil {
		server.Settings = make(map[string]interface{})
	}

	if versionText := detectFactorioVersion(server.Paths.FactorioBinary, bootstrap.GetConfig().GlibcCustom, bootstrap.GetConfig().GlibcLocation, bootstrap.GetConfig().GlibcLibLoc); versionText != "" {
		_ = server.Version.UnmarshalText([]byte(versionText))
		server.VersionLabel = versionText
	}

	baseModInfoFile := filepath.Join(server.Paths.BaseModDir, "info.json")
	if data, err := os.ReadFile(baseModInfoFile); err == nil {
		var modInfo ModInfo
		if err := json.Unmarshal(data, &modInfo); err == nil {
			server.BaseModVersion = modInfo.Version
		}
	}
	return nil
}

func buildInstancePaths(instanceRoot, versionDir string) InstancePaths {
	configDir := filepath.Join(instanceRoot, "config")
	modsDir := filepath.Join(instanceRoot, "mods")
	logsDir := filepath.Join(instanceRoot, "logs")
	binary := filepath.Join(versionDir, "bin", "x64", "factorio")
	return InstancePaths{
		Root:           instanceRoot,
		VersionDir:     versionDir,
		FactorioBinary: binary,
		BaseModDir:     filepath.Join(versionDir, "data", "base"),
		SavesDir:       filepath.Join(instanceRoot, "saves"),
		ModsDir:        modsDir,
		ModPackDir:     filepath.Join(instanceRoot, "mod_packs"),
		ConfigDir:      configDir,
		ConfigFile:     filepath.Join(configDir, "config.ini"),
		SettingsFile:   filepath.Join(configDir, "server-settings.json"),
		AdminFile:      filepath.Join(configDir, "server-adminlist.json"),
		LogsDir:        logsDir,
		ConsoleLogFile: filepath.Join(logsDir, "factorio-server-console.log"),
		FactorioLog:    filepath.Join(logsDir, "factorio-current.log"),
		ModSettingsDat: filepath.Join(modsDir, "mod-settings.dat"),
	}
}

func createInstanceDirs(paths InstancePaths) error {
	for _, dir := range []string{paths.Root, paths.SavesDir, paths.ModsDir, paths.ModPackDir, paths.ConfigDir, paths.LogsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func detectFactorioVersion(binary, glibcCustom, glibcLocation, glibcLibLoc string) string {
	if binary == "" {
		return ""
	}
	if _, err := os.Stat(binary); err != nil {
		return ""
	}

	var out []byte
	var err error
	if glibcCustom == "true" {
		out, err = exec.Command(glibcLocation, "--library-path", glibcLibLoc, binary, "--version").Output()
	} else {
		out, err = exec.Command(binary, "--version").Output()
	}
	if err != nil {
		return ""
	}
	reg := regexp.MustCompile(`Version.*?((\d+\.)?(\d+\.)?(\*|\d+)+)`)
	found := reg.FindStringSubmatch(string(out))
	if len(found) > 1 {
		return found[1]
	}
	return ""
}

func migrateLegacyInstall(factorioDir, versionDir string) error {
	if factorioDir == "" || versionDir == "" {
		return nil
	}
	if samePath(factorioDir, versionDir) {
		return nil
	}
	for _, name := range []string{"bin", "data", "doc"} {
		src := filepath.Join(factorioDir, name)
		dst := filepath.Join(versionDir, name)
		if err := movePathIfExists(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func movePathIfExists(src, dst string) error {
	if src == "" || dst == "" || samePath(src, dst) {
		return nil
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyPath(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst, info.Mode())
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
}

func parsePortRange(value string) (int, int) {
	parts := strings.SplitN(value, "-", 2)
	start := 34197
	end := 34220
	if len(parts) > 0 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			start = parsed
		}
	}
	if len(parts) > 1 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			end = parsed
		}
	}
	if end < start {
		end = start
	}
	return start, end
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
