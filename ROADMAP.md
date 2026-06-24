# FSM Roadmap

## ✅ Done
- [x] Fix mod folder cleanup on Docker volume
- [x] Fix save file parsing for Factorio 2.0
- [x] Read all mods with exact versions directly from save file
- [x] New endpoint to read mods from save without downloading
- [x] Selective mod sync with checkboxes (select/deselect all)
- [x] Real-time download progress via WebSocket
- [x] DLC mods auto-detected and grouped in UI with single toggle
- [x] DLC mods visible in installed mods list with enable/disable
- [x] Server start blocked while mod sync is in progress
- [x] i18n support (EN/RU/ZH)
- [x] Token-based authentication on factorio.com
- [x] Dynamic locale loading via HTTP backend
- [x] Download locale template from UI
- [x] Upload custom locale from UI (with overwrite confirmation)
- [x] Active language highlighted in language dialog
- [x] Portal link icon next to each mod in the list
- [x] Factorio installation via web UI (download & install)
- [x] Factorio version dropdown (stable/experimental) in Server Status
- [x] Warning when Factorio is not installed
- [x] FSM starts without Factorio binary (graceful degradation)
- [x] server-settings.json auto-created from example after install
- [x] Full UI translation coverage (all views and components)

## 🚧 In Progress
- [ ] Autostart toggle (UI done, backend wiring in progress)
- [ ] Save selected save file between page reloads
- [ ] Auto-locale generation via Google Translate API on first login

## 📋 Planned

### Locales
- [ ] Google Translate API integration for auto-locale generation
- [ ] Prompt user to generate locale if browser language has no match
- [ ] Force-generate locale button with language selector in UI

### Mods
- [ ] Auto-resolve mod dependencies when creating a new save
- [ ] Real byte-based progress bar for mod downloads
- [ ] Row highlight on hover in mod list

### Server
- [ ] Autostart — save to config and wire to FSM --autostart flag
- [ ] Remember selected save file (localStorage)
- [ ] Server name field
- [ ] Multi-version Factorio support via symlinks
  - [ ] Install multiple versions side by side
  - [ ] Switch active version via symlink
  - [ ] Delete old versions from UI
- [ ] Import server-settings.json from UI
- [ ] Multi-server support (future)
- [ ] Dual logs — Factorio server logs + FSM manager logs

### Authentication
- [ ] Show logged-in username on mod portal tab
- [ ] Refresh button for saved credentials
- [ ] Proper FSM login (registration form on first launch)
