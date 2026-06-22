# FSM Server Version Manager — Implementation Plan

**Date:** 2026-06-21
**Spec:** `docs/superpowers/specs/2026-06-21-server-version-manager-design.md`
**Branch:** `feat/server-version-manager`

---

## Context

Add a Server Version Manager to FSM that allows operators to view installed version, fetch available versions from the Factorio API, and download/install a selected version — all from the UI. The install requires server-off state, uses existing Mod Portal credentials for auth, and reports download progress via WebSocket.

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| T1: go.mod add xz dependency | None | Foundation — needed for compilation |
| T2: version_manager.go core logic | T1 | Uses xz package for extraction |
| T3: version_manager_test.go | T2 | Tests the core logic |
| T4: version_handler.go API handlers | T2 | Handlers call VersionManager methods |
| T5: routes.go backend API routes | T4 | Must reference handler functions |
| T6: routes.go frontend SPA route | None | Independent of other backend changes |
| T7: version_handler_test.go | T4, T5 | Tests the handler integration |
| T8: en/serverVersion.json | None | Independent of backend |
| T9: zh-CN/serverVersion.json | None | Independent of backend |
| T10: i18n.js registration | T8, T9 | Must import the new locale files |
| T11: server.js + socket.js API/WS | None | Independent of backend |
| T12: ServerVersion.jsx view | T11, T10, T8, T9 | Needs API methods and i18n strings |
| T13: App.jsx route registration | T12 | Must import ServerVersion component |
| T14: Layout.jsx sidebar link + i18n | T13 | Sidebar nav for the new page |
| T15: Build + verify | All | Final integration verification |

## Parallel Execution Graph

**Wave 1 (Start immediately — no inter-file dependencies):**
- T1: go.mod add `ulikunitz/xz`
- T8: Create `en/serverVersion.json`
- T9: Create `zh-CN/serverVersion.json`
- T11: Add API methods + WS subscription to server.js/socket.js

**Wave 2 (After Wave 1):**
- T2: `version_manager.go` — core logic (download, extract, install, progress)
- T6: Frontend SPA route in routes.go

**Wave 3 (After Wave 2 completes):**
- T4: `version_handler.go` — 3 API handler functions
- T5: Register backend routes in routes.go
- T10: Register i18n namespace in i18n.js

**Wave 4 (After Wave 3 completes):**
- T3: `version_manager_test.go`
- T7: `version_handler_test.go`

**Wave 5 (After Wave 4 completes):**
- T12: `ServerVersion.jsx` — full view component with progress bar

**Wave 6 (After Wave 5 completes):**
- T13: Add route in App.jsx
- T14: Add sidebar link in Layout.jsx + layout locale keys

**Wave 7 (After Wave 6 completes):**
- T15: Build frontend + backend, verify compilation

**Critical Path:** T1 → T2 → T4 → T5 → T12 → T13 → T14 → T15

## Tasks

### Task 1: go.mod — Add `ulikunitz/xz` dependency

**Description**: Add the pure-Go xz decompression library required by `extractTarXz()`.

**File**: `src/go.mod`

**Changes**:
- Add `github.com/ulikunitz/xz v0.5.12` to the `require` block
- Run `cd src && go mod tidy` to update `go.sum`

**Delegation Recommendation**:
- Category: `quick` — single-line dep addition
- Skills: none needed

**Depends On**: None

**Acceptance Criteria**:
- `go.sum` updated
- `go build ./...` passes

---

### Task 2: version_manager.go — Core backend logic

**Description**: New file containing `VersionManager` struct, `GetInstalledVersion()`, `FetchAvailableVersions()`, `DownloadAndInstall()`, `extractTarXz()`, `ProgressReader`, and `reportProgress()`.

**File**: `src/factorio/version_manager.go` (new)

**Key types** (per spec §2.3):
- `LatestReleasesResponse`, `LatestRelease`, `AvailableVersion`
- `ProgressReader` — io.Reader wrapper that reports download % to WebSocket room every ~5% delta or at 100%

**Key functions**:

| Function | Detail |
|---|---|
| `GetInstalledVersion()` | Runs `config.FactorioBinary --version`, regex-parses output (same pattern as `NewFactorioServer()` in server.go:150-167). Returns `Version`. |
| `FetchAvailableVersions()` | GET `https://factorio.com/api/latest-releases`, parse JSON. Generate candidate list from `2.0.0` up to `latest_experimental` by incrementing patch. Returns `[]AvailableVersion` with stable/experimental flags. |
| `DownloadAndInstall(v Version)` | 1. `credentials.Load()` → fail 403 if none. 2. Construct URL with username+token. 3. HTTP GET, stream to `/tmp/factorio-dl-<random>.tar.xz` with ProgressReader. 4. `extractTarXz()` over `config.FactorioDir`. 5. `defer os.Remove` temp. 6. Verify with GetInstalledVersion(). |
| `extractTarXz(r io.Reader, dest string)` | xz.NewReader → tar.NewReader. For each header: create dirs, write files, set 0755 on `bin/x64/factorio`. |
| `reportProgress(percent, stage, msg)` | Marshal JSON, send to `websocket.WebsocketHub.GetRoom("server_version").Send()` |

**Helper**: `IsUpdateAvailable(installed, latestStable Version) bool` — uses existing `Greater()`.

**WS Event format** (per spec §6.1):
```json
{"room": "server_version", "event": "install_progress", "data": {"percent": 45, "stage": "download", "message": "45.2 MB / 102.4 MB"}}
```

**Delegation Recommendation**:
- Category: `deep` — moderately complex Go with HTTP, xz, websocket integration
- Skills: none needed beyond system Go toolchain

**Depends On**: Task 1

**Acceptance Criteria**:
- Compiles with `go build ./...`
- `GetInstalledVersion()` returns a valid `Version` when pointed at a real binary
- Progress events sent to WebSocket room during download

---

### Task 3: version_manager_test.go — Tests for core logic

**Description**: Unit tests for version comparison, version range generation, and credential checking.

**File**: `src/factorio/version_manager_test.go` (new)

**Tests** (per spec §12.1):

| Test | Input | Expected |
|---|---|---|
| `TestGetInstalledVersion` | Real factorio binary at configured path | Valid Version parsed from --version |
| `TestFetchAvailableVersions` | Mock HTTP server returning latest-releases | []AvailableVersion with stable + experimental |
| `TestDownloadAndInstall` | Mock HTTP serving valid tar.xz | Binary extracted at FactorioDir |
| `TestInstallRequiresCredentials` | No factorio.auth file | Error returned |
| `TestInstallWithBadVersion` | Version "0.0.0" | Invalid version error |
| `TestIsUpdateAvailable` | (2.0.28, 2.0.32) | true |
| `TestIsUpdateAvailable_Equal` | (2.0.32, 2.0.32) | false |
| `TestVersionRangeGeneration` | stable=2.0.32, experimental=2.0.32 | List covers 2.0.0..2.0.32, 2.0.32 is both |

Use `testing.Short()` skip for tests needing external HTTP or real binary.

**Delegation Recommendation**:
- Category: `quick` — straightforward unit test patterns matching existing style
- Skills: none

**Depends On**: Task 2

**Acceptance Criteria**:
- `cd src && go test -short ./factorio/ -run TestVersion` passes

---

### Task 4: version_handler.go — API HTTP handlers

**Description**: Three handler functions following the existing pattern in `mod_portal_handler.go`: `defer WriteResponse` + `w.Header().Set("Content-Type", ...)` + explicit status codes.

**File**: `src/api/version_handler.go` (new)

**Handlers**:

**`CurrentVersionHandler`**:
- Calls `factorio.GetInstalledVersion()` (or reads from singleton `factorio.GetFactorioServer().Version`)
- Returns `{"version": "2.0.28", "string": "2.0.28"}`
- Error 500 on failure

**`AvailableVersionsHandler`**:
- Calls `factorio.FetchAvailableVersions()`
- Returns `{"stable": "2.0.32", "experimental": "2.0.32", "versions": [...]}`
- Error 502 on API fetch failure

**`InstallVersionHandler`**:
- Reads `{"version": "2.0.32"}` from body via `ReadFromRequestBody`
- Validates with `Version.UnmarshalText()`
- Gets previous version before installing
- Calls `factorio.DownloadAndInstall()`
- Returns `{"success": true, "version": "2.0.32", "previousVersion": "2.0.28", "installedAt": "RFC3339"}`
- Error 400 bad version, 403 no credentials, 500 install failure

**Delegation Recommendation**:
- Category: `deep` — request parsing + multiple error paths + WS integration
- Skills: none

**Depends On**: Task 2

**Acceptance Criteria**:
- Compiles
- Each handler returns correct JSON shape per spec §3

---

### Task 5: routes.go — Register backend API routes

**Description**: Add three new route entries to `apiRoutes` slice.

**File**: `src/api/routes.go`

**Changes to `apiRoutes`** — append three routes (before the closing `}`):

```go
{
    "CurrentVersion",
    "GET",
    "/server/version/current",
    CurrentVersionHandler,
    false,
}, {
    "AvailableVersions",
    "GET",
    "/server/version/available",
    AvailableVersionsHandler,
    false,
}, {
    "InstallVersion",
    "POST",
    "/server/version/install",
    InstallVersionHandler,
    true,  // requires server-off
},
```

**Delegation Recommendation**:
- Category: `quick` — simple slice append following existing pattern
- Skills: none

**Depends On**: Task 4

**Acceptance Criteria**:
- `go build ./...` compiles
- Routes listed in `NewRouter()` output when inspected

---

### Task 6: routes.go — Frontend SPA route

**Description**: Add frontend file-server route for `/server-version` alongside existing routes like `/saves`, `/mods`, `/logs`.

**File**: `src/api/routes.go`

**Change** — add after the `/logs` route block (around line 116):

```go
subRouter.Path("/server-version").
    Methods("GET").
    Name("Server Version").
    Handler(http.StripPrefix("/server-version", http.FileServer(http.Dir("./app/"))))
```

**Delegation Recommendation**:
- Category: `quick` — single route addition following exact existing pattern
- Skills: none

**Depends On**: None

**Acceptance Criteria**:
- Route registered in router
- Navigable to `/server-version` in browser

---

### Task 7: version_handler_test.go — Tests for API handlers

**Description**: Integration tests for the three version endpoints using existing `CallRoute` pattern from `mods_handler_test.go`.

**File**: `src/api/version_handler_test.go` (new)

**Tests** (per spec §12.1):

| Test | Method | Route | Expected |
|---|---|---|---|
| `TestCurrentVersion` | GET | `/api/server/version/current` | 200 + `{"version":"...", "string":"..."}` |
| `TestAvailableVersions` | GET | `/api/server/version/available` | 200 + version list (or skip in short mode) |
| `TestInstall_ServerRunning` | POST | `/api/server/version/install` | 423 Locked |
| `TestInstall_InvalidVersion` | POST | `/api/server/version/install` | 400 bad version |

**Delegation Recommendation**:
- Category: `quick` — simple handler tests following existing patterns
- Skills: none

**Depends On**: Task 4, Task 5

**Acceptance Criteria**:
- `cd src && go test -short ./api/ -run TestVersion` passes

---

### Task 8: en/serverVersion.json — English i18n strings

**File**: `ui/locales/en/serverVersion.json` (new)

Content per spec §10.1 — all keys as documented.

**Delegation Recommendation**:
- Category: `quick` — static JSON file
- Skills: none

**Depends On**: None

**Acceptance Criteria**:
- JSON is valid
- All keys from spec are present

---

### Task 9: zh-CN/serverVersion.json — Chinese i18n strings

**File**: `ui/locales/zh-CN/serverVersion.json` (new)

Content per spec §10.1 — all keys translated.

**Delegation Recommendation**:
- Category: `quick` — static JSON file
- Skills: none

**Depends On**: None

**Acceptance Criteria**:
- JSON is valid
- All keys match en version structure

---

### Task 10: i18n.js — Register new serverVersion namespace

**File**: `ui/i18n.js`

**Changes**:
- Import `enServerVersion` and `zhServerVersion`
- Add `serverVersion: enServerVersion` to `resources.en`
- Add `serverVersion: zhServerVersion` to `resources['zh-CN']`

**Delegation Recommendation**:
- Category: `quick` — 4-line addition following exact existing import/registration pattern
- Skills: none

**Depends On**: Task 8, Task 9

**Acceptance Criteria**:
- i18n loads without errors
- `t('title', { ns: 'serverVersion' })` returns the english title string

---

### Task 11: server.js + socket.js — API resource methods + WS subscription

**Files**: `ui/api/resources/server.js`, `ui/api/socket.js`

**server.js changes**:
- Add `currentVersion()` — `client.get('/api/server/version/current')`
- Add `availableVersions()` — `client.get('/api/server/version/available')`
- Add `installVersion(version)` — `client.post('/api/server/version/install', { version })`

**socket.js changes**:
- In `registerEventEmitter`/`unregisterEventEmitter`, add handler functions:
  ```js
  function versionSubscribeEvent() {
      socket.send(JSON.stringify({room_name: "", controls: {type: "subscribe", value: "server_version"}}));
  }
  function versionUnsubscribeEvent() {
      socket.send(JSON.stringify({room_name: "", controls: {type: "unsubscribe", value: "server_version"}}));
  }
  ```
- Register: `bus.on('version progress subscribe', versionSubscribeEvent);`
- Unregister: `bus.off('version progress subscribe', versionSubscribeEvent);` + unsubscribe

**Delegation Recommendation**:
- Category: `quick` — straightforward method additions following patterns
- Skills: none

**Depends On**: None

**Acceptance Criteria**:
- `server.installVersion('2.0.32')` sends POST with correct body
- `socket.emit('version progress subscribe')` sends WS subscribe for `server_version` room

---

### Task 12: ServerVersion.jsx — New view component

**Description**: Full React component implementing the UI layout from spec §4.

**File**: `ui/App/views/ServerVersion.jsx` (new)

**States covered** (per spec §4.3):

| State | UI |
|---|---|
| Loading | "Loading..." in each section |
| Loaded, up-to-date | Green checkmark + "Up to date" |
| Loaded, update available | "Update Available" badge + big CTA button |
| Installing | Progress bar (Tailwind) + disabled buttons |
| Error | Red error banner |
| No credentials | Link to Mod Portal settings page |

**Key implementation**:
- `useTranslation('serverVersion')`
- `useEffect` fetch current + available versions on mount
- Subscribe to `server_version` WS room for progress events
- `handleInstall(version)` — calls `server.installVersion`, shows progress
- Version list with stable/experimental labels
- Confirmation via `window.confirm` (or existing `ConfirmDialog` component)
- Progress bar: `<div className="bg-orange h-4 rounded" style={{ width: progress.percent + '%' }}>`

**Delegation Recommendation**:
- Category: `visual-engineering` — new React view with i18n and WS integration
- Skills: none specific

**Depends On**: Task 10, Task 11

**Acceptance Criteria**:
- Component renders all states without errors
- Progress bar appears and updates during install
- Error display matches spec §7.3

---

### Task 13: App.jsx — Route registration

**File**: `ui/App/App.jsx`

**Changes**:
- Add `import ServerVersion from "./views/ServerVersion";`
- Add route:
  ```jsx
  <Route path="server-version" element={<ServerVersion serverStatus={serverStatus} />} />
  ```

**Delegation Recommendation**:
- Category: `quick` — single import + single line route
- Skills: none

**Depends On**: Task 12

**Acceptance Criteria**:
- `/server-version` renders ServerVersion component

---

### Task 14: Layout.jsx — Sidebar link + layout i18n

**Files**: `ui/App/components/Layout.jsx`, `ui/locales/en/layout.json`, `ui/locales/zh-CN/layout.json`

**Layout.jsx changes**:
- Add link after `<Link to="/logs">`:
  ```jsx
  <Link to="/server-version">{t('linkServerVersion', { ns: 'layout' })}</Link>
  ```
- Move `last={true}` from `linkLogs` to this new link

**layout.json changes**:
- `en/layout.json`: add `"linkServerVersion": "Server Version"`
- `zh-CN/layout.json`: add `"linkServerVersion": "服务器版本"`

**Delegation Recommendation**:
- Category: `quick` — 1 link + 2 JSON keys
- Skills: none

**Depends On**: Task 13

**Acceptance Criteria**:
- Sidebar shows "Server Version" link in Server Management section
- Navigation to /server-version works from sidebar

---

### Task 15: Build + Verify

**Description**: Build both frontend and backend, verify compilation.

**Steps**:
1. `npm install`
2. `npm run build`
3. `cd src && go build -o ../factorio-server-manager/factorio-server-manager .`
4. `cd src && go vet ./...`
5. Verify `app/bundle.js` and binary exist and are non-empty

**Delegation Recommendation**:
- Category: `quick` — run build commands, check output
- Skills: none

**Depends On**: All tasks

**Acceptance Criteria**:
- Frontend builds with no errors
- Backend builds with no errors
- All `go vet` checks pass

---

## Commit Strategy

1. After writing this plan:
   ```
   文档: 添加服务器版本管理器实现计划
   ```

2. After full implementation (single atomic commit):
   ```
   功能: 实现服务器版本管理器
   ```

   If intermediate commits are useful during development, squash to a single commit before final.

---

## Success Criteria

1. All tasks pass their acceptance criteria
2. `cd src && go build ./...` succeeds
3. `npm run build` succeeds
4. `cd src && go test -short ./...` passes
5. Three API endpoints return correct JSON per spec §3
6. Frontend renders all states (loading, loaded, installing, error)
7. Progress events flow backend → WS room → frontend progress bar
8. Install blocked when server running (423) or no credentials (403)
