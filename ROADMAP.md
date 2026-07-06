# FSM Roadmap

## ✅ Done

### Core
- Fix: mod folder cleanup on Docker volume
- Fix: save file parsing for Factorio 2.0
- Fix: server-settings.json corrupted to null on server start
- Fix: GetServerSettings read from memory instead of file
- Fix: UpdateServerSettings overwrote file with form-only data (now merges)
- Fix: autostart saved to conf.json
- Fix: Dockerfile warnings (FromAsCasing, LegacyKeyValueFormat)
- Fix: ca-certificates missing in Docker image (mod portal SSL errors)
- Fix: Error starting Factorio server: %!s(<nil>) — nil error logging fixed
- Fix: server status WebSocket not updating on first server start
- Fix: .ZIP uppercase extension breaks Factorio save loading — normalized to .zip
- Fix: save path traversal protection (thanks @TheCoolestPaul)
- Fix: phantom server created on clean install — migrateLegacy now checks for real legacy data
- Fix: "FactorioDir not found" error on clean start (ModStartUp checks empty path)

### Multi-server
- Add: multi-server support — server registry, per-server instances (thanks @TheCoolestPaul)
- Add: server list dashboard with cards (start/stop/kill/save/manage/delete)
- Add: per-server isolated paths (/opt/factorio-server/instances/N/)
- Add: shared Factorio version cache (/opt/factorio-server/versions/)
- Add: legacy single-server migration to server ID 1 on first start
- Add: gap-filling server ID algorithm — reuses freed IDs
- Add: inline server card editing — IP, port, version, save without leaving dashboard
- Add: version_channel field — persists stable/experimental choice, survives loadMetadata()
- Fix: Factorio version not applied after server creation — PATCH now calls EnsureServerVersion
- Fix: CreateServer now installs Factorio immediately (not waiting for first start)
- Fix: deleting server 1 broke navigation — RedirectToServerScoped uses first real server
- Fix: hardcoded /servers/1/... redirects replaced with dynamic resolution
- Add: ServerScopeHeader dropdown — switch servers without going back to Controls
- Add: ServersContext — global server state with polling and reconcileAfterAction (thanks @TheCoolestPaul)
- Add: no-servers placeholder with link to create first server
- Add: Console nav button disabled when server is stopped

### Version management
- Add: Factorio installation via web UI (download & install)
- Add: /api/servers/next — preview next server ID, name and port before creation
- Add: /api/versions/installed — list installed Factorio versions
- Add: /api/versions/downloaded — list cached .tar.xz archives
- Add: DELETE /api/versions/downloaded/{version} — delete cached archive
- Add: DELETE /api/versions/installed/{version} — delete installed version (checks if in use)
- Add: create server form with per-version Download/Create/trash buttons
- Add: trash expands to zip/installed/all (all requires confirmation)
- Add: version status badge (installed · zip cached / not installed / zip cached · not installed)
- Add: free-text version input with normalizer (2076 → 2.0.76, auto-detects channel)
- Add: responsive create form — 3 fixed fields + version/buttons wrap on mobile

### Mods & Mod Packs
- Add: read all mods with exact versions directly from save file
- Add: new endpoint to read mods from save without downloading
- Add: selective mod sync with checkboxes (select/deselect all)
- Add: real-time download progress via WebSocket
- Add: DLC mods auto-detected and grouped in UI with single toggle
- Add: DLC mods visible in installed mods list with enable/disable
- Add: server start blocked while mod sync is in progress
- Add: Mod Options moved into Mods page as a tab (removed separate menu item)
- Add: mod list sorting by name and enabled state
- Fix: modpack create/delete uses per-server instance paths (not legacy global path)
- Add: modpack panel hidden when empty, shown only when packs exist
- Add: modpack create button moved to Mods panel actions
- Add: modpack share link — public /share/{modpack} endpoint, no auth required
- Add: copy share link button on each mod pack

### Saves
- Add: save list sorting by name, date, size (default: date DESC)
- Fix: save .ZIP extension normalized to .zip (Factorio binary requirement)
- Fix: save path traversal protection (thanks @TheCoolestPaul)

### UI/UX
- Add: i18n support (EN/RU/ZH) (inspired by @BAYUNZIYUE)
- Add: dynamic locale loading via HTTP backend
- Add: download/upload custom locale from UI
- Add: active language highlighted in language dialog
- Add: portal link icon next to each mod (thanks @TheCoolestPaul)
- Add: row highlight on hover in mod list (thanks @TheCoolestPaul)
- Add: token-based authentication on factorio.com
- Add: validate Factorio mod portal credentials on login (thanks @TheCoolestPaul)
- Add: real byte-based progress bar for Factorio installation
- Add: real byte-based progress bar for mod sync downloads + cancel
- Add: autostart toggle
- Add: FSM logs page
- Add: Game Settings warning when config.ini not available
- Add: hardcoded default server-settings.json with all fields and comments
- Add: custom default config support via /opt/fsm-data/default-server-settings.json
- Add: full UI translation coverage
- Add: per-server scoped API routes
- Fix: controls i18n — autostart, install_factorio, factorio_not_installed keys

## 🚧 In Progress

### Mods
- Mod pack creation with mod selection (checkboxes, not copy-all)
- Fix: modpack operations fully scoped to server instance (WIP — using DefaultServer() as interim)

### Server
- After Factorio install: compare example config with current, prompt to sync new fields
- Remember selected save file between page reloads

### Locales
- Translation coverage improvements (new multi-server pages)

## 📋 Planned

### Server
- Global saves/mods view across all servers with server column + save version
- Copy/move saves between server instances
- Docker port mapping validation — warn if server port not exposed
- After Factorio install: sync new fields from example config (dialog)
- Favicon

### Mods
- Mod pack creation with selective mod picking (checkboxes)

### Authentication
- Show logged-in username on mod portal tab
- Proper FSM login (registration form on first launch)

### UI
- Notifications on server start/stop (WebSocket flash)
- Upload save progress bar
- Dual logs — Factorio server logs + FSM manager logs in one view
- Public URL setting for share links (for servers behind NAT)
