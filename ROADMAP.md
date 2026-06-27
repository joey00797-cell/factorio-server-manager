# FSM Roadmap

## ✅ Done
- [x] Fix: mod folder cleanup on Docker volume
- [x] Fix: save file parsing for Factorio 2.0
- [x] Add: read all mods with exact versions directly from save file
- [x] Add: new endpoint to read mods from save without downloading
- [x] Add: selective mod sync with checkboxes (select/deselect all)
- [x] Add: real-time download progress via WebSocket
- [x] Add: DLC mods auto-detected and grouped in UI with single toggle
- [x] Add: DLC mods visible in installed mods list with enable/disable
- [x] Add: server start blocked while mod sync is in progress
- [x] Add: i18n support (EN/RU/ZH) (inspired by @BAYUNZIYUE)
- [x] Add: token-based authentication on factorio.com
- [x] Add: validate Factorio mod portal credentials on login (thanks @TheCoolestPaul)
- [x] Add: dynamic locale loading via HTTP backend (inspired by @BAYUNZIYUE)
- [x] Add: download locale template from UI
- [x] Add: upload custom locale from UI (with overwrite confirmation)
- [x] Add: active language highlighted in language dialog
- [x] Add: portal link icon next to each mod in the list (thanks @TheCoolestPaul)
- [x] Add: Factorio installation via web UI (download & install)
- [x] Add: Factorio version dropdown (stable/experimental) in Server Status
- [x] Add: warning when Factorio is not installed
- [x] Add: FSM starts without Factorio binary (graceful degradation)
- [x] Add: autostart toggle (saved to conf.json, UI toggle in Controls)
- [x] Add: hardcoded default server-settings.json with all fields and comments
- [x] Add: custom default config support via /opt/fsm-data/default-server-settings.json
- [x] Add: diff logging in UpdateServerSettings (only changed fields logged)
- [x] Add: full UI translation coverage (all views and components)
- [x] Add: row highlight on hover in mod list (thanks @TheCoolestPaul)
- [x] Fix: server-settings.json corrupted to null on server start
- [x] Fix: GetServerSettings read from memory instead of file
- [x] Fix: UpdateServerSettings overwrote file with form-only data (now merges)
- [x] Fix: react-hook-form Input/Checkbox missing name prop (values not submitted)
- [x] Fix: autostart saved to conf.json
## 🚧 In Progress
### Server
- [x] Autostart toggle (saved to conf.json, UI toggle in Controls)
- [ ] After Factorio install: compare example config with current, prompt to sync new fields
- [x] Add: Game Settings warning when config.ini not available
- [x] Fix: controls i18n — autostart, install_factorio, factorio_not_installed keys
- [x] Fix: Dockerfile warnings (FromAsCasing, LegacyKeyValueFormat)
- [ ] Remember selected save file between page reloads
- [ ] Factorio installation progress bar
- [ ] Multi-version Factorio support via symlinks
- [ ] Add server page (first step toward multi-server: local or remote via RCON)
- [ ] Import server-settings.json from UI
- [ ] Server name field

### Locales
- [ ] Auto-locale generation via Google Translate API on first login
- [ ] Translation coverage improvements (new features may introduce untranslated strings)
- [ ]  - One of them is the entry point instead of the locale. I know) just select the locale in the menu

## 📋 Planned

### Mods
- [ ] Auto-resolve mod dependencies when creating a new save
- [ ] Real byte-based progress bar for mod downloads

### Server
- [ ] Multi-server support:
  - [ ] New server list page (empty state + "Create server" button)
  - [ ] Server instance architecture (/opt/factorio-server/downloads/ + instances/)
  - [ ] Download version → cached tar.xz, status: DOWNLOADED
  - [ ] Create server → extract to instance folder, status: INSTALLED  
  - [ ] Server card UI (name, IP, port, version, save, start/stop/kill)
  - [ ] Limit to 1 server for now ("Multi-server coming soon" on second create)
- [ ] Dual logs — Factorio server logs + FSM manager logs
- [ ] Extended startup logging for debugging server-settings.json null bug

### Authentication
- [ ] Show logged-in username on mod portal tab
- [ ] Refresh button for saved credentials
- [ ] Proper FSM login (registration form on first launch)
