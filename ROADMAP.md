# FSM Roadmap

## ✅ Done

### v6.6 (2026-08-26)
- Add: save backups — scheduled backups with interval and retention
- Add: backup deduplication — skip if save unchanged
- Add: backup pin — protect backup from pruning
- Add: backup rename — inline rename in UI
- Add: backup restore — restore to any server instance
- Add: backup download
- Fix: entrypoint pre-flight checks (Docker, permissions, setup.conf)
- Fix: setup wizard strips http:// prefix from HOST_IP input
- Fix: Windows build disabled by default (BUILD_WINDOWS=1 to enable)
- Add: setup.sh generates setup.conf and passes FSM_SERVER_IP
- Chore: remove unused legacy files (Console, FsmLogs, Logs, ModOptions, UploadMod views)

### v6.5 (2026-08-18)
- Add: RCON watchdog per server — configurable interval, auto-reconnect
- Add: /api/info endpoint — returns real server IP
- Add: real server IP shown in UI (Bind IP field, logs)
- Add: DLC mods expandable group — toggle each DLC component individually
- Add: upload button for mod-settings.dat with flash notification
- Add: docker/entrypoint.sh — single source of truth (removed ~/fsm-image/entrypoint.sh)

### v6.3–v6.4
- Add: Mod Library system — centralized mod storage, GORM models, manifest, preview/apply
- Add: DLC mods auto-created as assets (space-age, elevated-rails, quality)
- Add: SyncModsFromSave downloads to mod-library and registers in DB
- Add: mod delete flow — trash icon marks to_delete, Apply removes from library + disk
- Add: Load Mods from Save with WebSocket progress
- Add: Mod Options tab in Mods page (upload mod-settings.dat)
- Add: mod list sorting by name and enabled state
- Fix: modpack operations scoped to server instance

### Core
- Fix: mod folder cleanup on Docker volume
- Fix: save file parsing for Factorio 2.0
- Fix: server-settings.json corrupted to null on server start
- Fix: GetServerSettings read from memory instead of file
- Fix: UpdateServerSettings overwrote file with form-only data (now merges)
- Fix: autostart saved to conf.json
- Fix: Dockerfile warnings
- Fix: ca-certificates missing in Docker image (mod portal SSL errors)
- Fix: server status WebSocket not updating on first server start
- Fix: .ZIP uppercase extension normalized to .zip
- Fix: save path traversal protection (thanks @TheCoolestPaul)
- Fix: phantom server on clean install
- Fix: "FactorioDir not found" on clean start

### Multi-server
- Add: multi-server support (thanks @TheCoolestPaul)
- Add: server list dashboard with cards
- Add: per-server isolated paths (/opt/factorio-server/instances/N/)
- Add: shared Factorio version cache
- Add: legacy single-server migration
- Add: inline server card editing
- Add: ServerScopeHeader dropdown
- Add: ServersContext with polling (thanks @TheCoolestPaul)
- Fix: various multi-server routing and navigation fixes

### Version management
- Add: Factorio installation via web UI
- Add: version management UI with download/install/delete
- Add: free-text version input with normalizer
- Add: version status badges

### Saves
- Add: save list sorting by name, date, size
- Add: save backups with scheduling, deduplication, pin, rename, restore

### UI/UX
- Add: i18n support EN/RU/ZH
- Add: dynamic locale loading
- Add: flash notifications on server start/stop/kill/save
- Add: server Starting... intermediate status (rcon_connected)
- Add: unified Logs page with tabs
- Add: auto-scroll toggle on log views
- Add: favicon
- Add: real progress bars for installation and mod sync

## 🚧 In Progress

### Mods
- Refactor: remove legacy parallel mod system (mod_Mods.go, mods.go, ModSimpleList)
  — old system still lives alongside new manifest system

## 📋 Planned

### Mods
- mod-list.json import with manifest preview/diff UI
- Mod portal search in ModLibrary

### UI
- Upload save progress bar
- Public URL setting for share links (for servers behind NAT)

### Authentication
- Proper FSM login (registration form on first launch)

### Technical debt
- Replace deprecated ioutil with os/io
- Remove GetFactorioServer() / instantiated legacy single-server globals
