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
- [x] i18n support (EN/RU/ZH) (inspired by @BAYUNZIYUE)
- [x] Token-based authentication on factorio.com
- [x] Validate Factorio mod portal credentials on login (thanks @TheCoolestPaul)
- [x] Dynamic locale loading via HTTP backend (inspired by @BAYUNZIYUE)
- [x] Download locale template from UI
- [x] Upload custom locale from UI (with overwrite confirmation)
- [x] Active language highlighted in language dialog
- [x] Portal link icon next to each mod in the list (thanks @TheCoolestPaul)
- [x] Factorio installation via web UI (download & install, auto server-settings.json)
- [x] Factorio version dropdown (stable/experimental) in Server Status
- [x] Warning when Factorio is not installed
- [x] FSM starts without Factorio binary (graceful degradation)
- [x] Full UI translation coverage (all views and components)

## 🚧 In Progress
### Server
- [ ] Autostart toggle (UI done, backend wiring in progress)
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
- [ ] Row highlight on hover in mod list

### Server
- [ ] Multi-server support (future)
- [ ] Dual logs — Factorio server logs + FSM manager logs

### Authentication
- [ ] Show logged-in username on mod portal tab
- [ ] Refresh button for saved credentials
- [ ] Proper FSM login (registration form on first launch)
