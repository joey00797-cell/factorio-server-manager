# Changelog

Changes of this fork. History of the original project: [CHANGELOG-upstream.md](CHANGELOG-upstream.md). Detailed items: [ROADMAP.md](ROADMAP.md).

## [6.7.3] - 2026-11-05
### Fixed
- Dark theme contrast: server header badges, server card labels, disabled role field; empty email shows "—" (#15)
- Interface strings that ignored the language files (#13): server settings labels and comments, mod-settings note, Factorio version badge, Sign In, Sync menu, bulk mod buttons, presets and saves, dialogs
- Apply preview warnings and sync errors are now built from codes, so they follow the selected language

### Added
- `tools/check-i18n.py`: compares locale keys, finds keys missing in locales and hard-coded English text
- Missing Russian strings filled in (Sign In, language name, visibility, presets)

## [6.7.2] - 2026-10-30
### Added
- Mods: Enable all / Disable all / Remove all (from server or from library)
- Progress bars for portal downloads and dependency sync
- Version dialog: Esc closes it, shows why a version is incompatible
- Chinese translation

### Fixed
- Mod dependency parsing: `!`, `?`, `(?)`, `~`, `+` and `base >= x`; mods are chosen by the server's Factorio version
- Duplicate mod versions left in the manifest after a version switch
- Server crash on a broken mod zip (`log.Fatal` replaced by a skip and a log line)
- Blank page when deleting a Factorio version that is in use
- Docker: the container exits with the real FSM exit code

## [6.7.1] - 2026-10-28
### Added
- Native Linux install without Docker: one-line installer/updater (`install.sh`) and `start.sh` (dependency check, first-run setup, port check, update hint)

## [6.7.0] - 2026-10-27
First master release of the fork.

### Added
- Windows build without Docker: `start.bat` / `start.ps1`, `prepare-factorio` (Steam client, client zip or manual path), ready banner with the initial admin password
- Docker setup helper: macvlan question first, host IP/interface/gateway autodetection (`--network host`), docker-compose for Docker Desktop
- Server Factorio version is pinned; stab/exp resolve only on an explicit change
- Help tab: quick start, how-to, danger zone (wipe / delete server, optional backups wipe)
- Sync modes Match save / Add from save
- Translations EN/RU for saves, mods, presets, servers

### Fixed
- Duplicate mod versions in mod-list.json and the permanent Apply diff
- Web port other than 80 in the Docker helper
- F5 white page on nested routes
- "broken pipe" and "record not found" noise in logs

### Changed
- Removed unreachable legacy code

## [6.3 - 6.6]
Multi-server, mod library, save backups, watchdog, version management. See [ROADMAP.md](ROADMAP.md).
