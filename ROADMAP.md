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
- [x] Add: Game Settings warning when config.ini not available
- [x] Add: real byte-based progress bar for Factorio installation (MB display)
- [x] Add: real byte-based progress bar for mod sync downloads (MB display + cancel)
- [x] Add: multi-server support — server registry, per-server instances (thanks @TheCoolestPaul)
- [x] Add: server list dashboard with cards (start/stop/kill/save/manage/delete)
- [x] Add: per-server isolated paths (/opt/factorio-server/instances/N/)
- [x] Add: shared Factorio version cache (/opt/factorio-server/versions/)
- [x] Add: legacy single-server migration to server ID 1 on first start
- [x] Add: editable server network config (bind IP, port, autostart per server)
- [x] Add: FSM logs page (separate from Factorio server logs)
- [x] Add: per-server scoped API routes
- [x] Add: mod options page per server (raw base64 mod-settings.dat editor)
- [x] Fix: server-settings.json corrupted to null on server start
- [x] Fix: GetServerSettings read from memory instead of file
- [x] Fix: UpdateServerSettings overwrote file with form-only data (now merges)
- [x] Fix: react-hook-form Input/Checkbox missing name prop (values not submitted)
- [x] Fix: autostart saved to conf.json
- [x] Fix: controls i18n — autostart, install_factorio, factorio_not_installed keys
- [x] Fix: Dockerfile warnings (FromAsCasing, LegacyKeyValueFormat)
- [x] Fix: ca-certificates missing in Docker image (mod portal SSL errors)

## 🚧 In Progress

### Server

- [ ] Factorio installation progress bar (WebSocket) — перенести в мультисервер архитектуру
- [ ] After Factorio install: compare example config with current, prompt to sync new fields
- [ ] Fix: server status WebSocket not updating on first server start (requires F5)
- [ ] Fix: Error starting Factorio server: %!s(<nil>) — кривое логирование nil error
- [ ] Remember selected save file between page reloads

### Mods

- [ ] Fix: mod sync compatibility with new multi-server architecture
- [ ] Auto-resolve mod dependencies when creating a new save

### Locales

- [ ] Auto-locale generation via Google Translate API on first login
- [ ] Translation coverage improvements (new multi-server pages)

## 📋 Planned

### Server

- [ ] Import server-settings.json from UI
- [ ] After Factorio install: sync new fields from example config (dialog)
- [ ] Favicon

### Authentication

- [ ] Show logged-in username on mod portal tab
- [ ] Refresh button for saved credentials
- [ ] Proper FSM login (registration form on first launch)

### UI

- [ ] Notifications on server start/stop (WebSocket flash)
- [ ] Upload save progress bar
- [ ] Dual logs — Factorio server logs + FSM manager logs in one view
