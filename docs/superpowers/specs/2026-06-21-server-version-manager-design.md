# FSM Server Version Manager Design Spec

**Date:** 2026-06-21
**Project:** factorio-server-manager-joey
**Branch:** feat/server-version-manager
**Status:** Draft / Approved for Implementation

---

## 1. Overview and Goals

### 1.1 Problem

The Factorio Server Manager (FSM) currently assumes the Factorio headless server binary is pre-installed and has zero version management capability. Operators must manually download, extract, and replace the binary when updating. This adds friction especially when Wube releases new stable builds (every few weeks) and experimental builds (more frequently).

### 1.2 Goals

- Allow operators to view the currently installed Factorio version from the UI.
- Fetch the list of available versions from the official Factorio API.
- Download and install a selected version with one click, replacing the existing binary in place.
- Reuse the existing Mod Portal authentication (token + username) for download authorization.
- Block installation while the Factorio server is running (fail-safe).
- Show download progress to give feedback during multi-second downloads.
- Fully localized in English and Simplified Chinese via the existing i18n framework.

### 1.3 Non-Goals

- Version co-existence / side-by-side installations.
- Custom download mirrors or proxy support.
- Automatic update polling or notifications.
- Docker image version management.
- Windows support for the download/install flow (the Factorio download URL is linux64; Windows users already manage their own updates).

---

## 2. Architecture

### 2.1 New Package: `factorio/version_manager.go`

A new file `src/factorio/version_manager.go` contains all backend logic. No changes to existing files beyond adding route registrations.

```go
package factorio

import (
    "archive/tar"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
    "github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
    "github.com/ulikunitz/xz"
)
```

**Note on xz decompression:** Go's standard library does not include an xz decoder. Use `github.com/ulikunitz/xz` (pure Go, no CGo). Add to go.mod:

```
require github.com/ulikunitz/xz v0.5.12
```

### 2.2 Download/Install Flow

```
POST /api/server/version/install {version: "2.0.32"}
   |
   +-- 1. Verify server is stopped (ServerOffMiddleware)
   +-- 2. Load credentials (username + token from factorio.auth)
   +-- 3. Download: https://factorio.com/get-download/{version}/headless/linux64?username=<u>&token=<t>
   +-- 4. Stream to temp file: /tmp/factorio-dl-<random>.tar.xz
   +-- 5. Send progress via websocket room "server_version"
   +-- 6. Extract tar.xz: the archive contains bin/x64/factorio + data/ + doc-html/ etc.
   +-- 7. Overwrite in-place: extract into config.FactorioDir (updating bin/x64/factorio, data/, etc.)
   +-- 8. Clean up temp files
   +-- 9. Re-read installed version (run factorio --version on new binary)
   +-- 10. Return {success: true, version: "2.0.32"}
```

In-place override means the tarball is extracted directly on top of `config.FactorioDir`. This works because:

- The headless tarball contains the same directory layout as a normal installation.
- The `data/` directory is updated alongside the binary.
- Saves, mods, and config are outside `config.FactorioDir` (they live in `config.FactorioSavesDir`, `config.FactorioModsDir`, and `config.FactorioConfigDir` respectively).

### 2.3 Version Data Structures

```go
// VersionManager provides download/install operations
type VersionManager struct{}

// LatestReleasesResponse matches the factorio.com/api/latest-releases endpoint
type LatestReleasesResponse struct {
    Stable       LatestRelease `json:"stable"`
    Experimental LatestRelease `json:"experimental"`
}

type LatestRelease struct {
    Alpha    Version `json:"alpha"`
    Demo     Version `json:"demo"`
    Headless Version `json:"headless"`
}

// AvailableVersion is used for the /available API response
type AvailableVersion struct {
    Version      Version `json:"version"`
    Stable       bool    `json:"stable"`
    Experimental bool    `json:"experimental"`
}
```

### 2.4 Key Functions

| Function | Purpose |
|---|---|
| `GetInstalledVersion() (Version, error)` | Runs `factorio --version`, parses output. Reuses existing logic from `NewFactorioServer()`. |
| `FetchAvailableVersions() ([]AvailableVersion, error)` | Calls `https://factorio.com/api/latest-releases`, builds candidate list. |
| `DownloadAndInstall(version Version) error` | Full download, extract, replace flow. Sends progress events to websocket room. |
| `extractTarXz(r io.Reader, dest string) error` | Decompresses xz, extracts tar to dest dir, sets executable bit on binary. |

---

## 3. API Design

All three endpoints are behind `/api/server/version/` and require authentication (via `AuthMiddleware`). The install endpoint requires server-off (via `ServerOffMiddleware`).

### 3.1 GET /api/server/version/current

Returns the currently installed version.

**Response 200:**
```json
{
    "version": "2.0.28",
    "string": "2.0.28"
}
```

**Error 500:**
```json
{
    "error": "Could not detect installed Factorio version"
}
```

### 3.2 GET /api/server/version/available

Fetches the list of available versions from the Factorio API.

**Response 200:**
```json
{
    "stable": "2.0.32",
    "experimental": "2.0.32",
    "versions": [
        {"version": "2.0.32", "stable": true, "experimental": true},
        {"version": "2.0.31", "stable": true, "experimental": false},
        {"version": "2.0.30", "stable": false, "experimental": true}
    ]
}
```

Note: The factorio API's `latest-releases` endpoint returns only the single latest stable and experimental version. To build a full list, the spec defines a **static version range** approach for v1:

**Version range strategy (v1):**
- Fetch `latest-releases` to get `latest_stable` and `latest_experimental`.
- Generate a candidate list from `2.0.0` up to `latest_experimental` by incrementing the patch/minor numbers.
- This avoids hitting the mod portal API for every version.
- The range covers all versions >= 2.0 since older installs are rare and the format has been stable since 2.0.

**Future enhancement (v2):** Parse the factorio.com download page or a community update feed for historical releases.

**Error 502:**
```json
{
    "error": "Failed to fetch version list from Factorio API"
}
```

### 3.3 POST /api/server/version/install

Downloads and installs a specific version. Requires `ServerOffMiddleware`.

**Request:**
```json
{
    "version": "2.0.32"
}
```

**Response 200:**
```json
{
    "success": true,
    "version": "2.0.32",
    "previousVersion": "2.0.28",
    "installedAt": "2026-06-21T10:30:00Z"
}
```

**Error 400 (bad version):**
```json
{
    "error": "Invalid version string"
}
```

**Error 403 (no auth):**
```json
{
    "error": "Factorio.com credentials required. Log in via Mod Portal settings first."
}
```

**Error 423 (server running):**
```json
{
    "error": "Factorio server running. Stop the server before updating."
}
```

**Error 500 (install failed):**
```json
{
    "error": "Download failed: connection timeout"
}
```

### 3.4 Route Registration

Add to `src/api/routes.go` in the `apiRoutes` slice:

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

### 3.5 Handler File: `src/api/version_handler.go`

New handler file following the pattern in `mod_portal_handler.go`:

```go
package api

func CurrentVersionHandler(w http.ResponseWriter, r *http.Request) { ... }
func AvailableVersionsHandler(w http.ResponseWriter, r *http.Request) { ... }
func InstallVersionHandler(w http.ResponseWriter, r *http.Request) { ... }
```

Each handler follows the existing `defer WriteResponse` + `w.Header().Set("Content-Type", ...)` pattern.

---

## 4. Frontend Component Design

### 4.1 New View: `ui/App/views/ServerVersion.jsx`

```jsx
import React, { useEffect, useState } from "react";
import { useTranslation } from 'react-i18next';
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import socket from "../../api/socket";

const ServerVersion = ({ serverStatus }) => {
    const { t } = useTranslation('serverVersion');

    const [installedVersion, setInstalledVersion] = useState(null);
    const [availableVersions, setAvailableVersions] = useState([]);
    const [stableVersion, setStableVersion] = useState(null);
    const [isInstalling, setIsInstalling] = useState(false);
    const [progress, setProgress] = useState(null);
    const [error, setError] = useState(null);

    useEffect(() => {
        fetchCurrentVersion();
        fetchAvailableVersions();
    }, []);

    useEffect(() => {
        socket.emit('subscribe', 'server_version');
        socket.on('server_version', data => {
            setProgress(data);
        });
        return () => {
            socket.off('server_version');
        };
    }, []);

    const handleInstall = async (version) => {
        setIsInstalling(true);
        setError(null);
        setProgress({ percent: 0, status: 'downloading' });
        try {
            const result = await server.installVersion(version);
            setInstalledVersion(result.version);
            setProgress(null);
        } catch (err) {
            setError(err.message);
            setProgress(null);
        } finally {
            setIsInstalling(false);
        }
    };

    const isUpdateAvailable = installedVersion && stableVersion &&
        installedVersion !== stableVersion;

    // Render logic
};
```

### 4.2 UI Layout (matching Controls Panel pattern)

```
+---------------------------------------------+
|  Server Version              (Panel title)   |
+---------------------------------------------+
|  Installed Version                          |
|  +---------------------------------------+  |
|  | Current: 2.0.28                       |  |
|  | Status: Update Available              |  |
|  |   (Update to 2.0.32)                  |  |
|  +---------------------------------------+  |
|                                             |
|  Available Versions                         |
|  +---------------------------------------+  |
|  | o 2.0.32 (latest stable)   [Install]  |  |
|  | o 2.0.31                  [Install]   |  |
|  | o 2.0.30 (experimental)   [Install]   |  |
|  +---------------------------------------+  |
|                                             |
|  [Update to 2.0.32]   big CTA button        |
|    only visible when update available        |
+---------------------------------------------+
```

### 4.3 State Machine

| State | Condition | UI |
|---|---|---|
| Loading | versions not yet fetched | "Loading..." spinner in each section |
| Loaded, up-to-date | installed == stable latest | Green checkmark, "Up to date" |
| Loaded, update available | installed < stable latest | "Update Available" badge, CTA button |
| Installing | isInstalling = true | Progress bar, disabled buttons |
| Error | error != null | Error banner with message |
| No credentials | 403 on install attempt | "Login required" link to Mod Portal |

### 4.4 API Resource: `ui/api/resources/server.js`

Add method to existing server resource:

```js
const ServerVersionResource = {
    current() {
        return api.get('/server/version/current');
    },
    available() {
        return api.get('/server/version/available');
    },
    install(version) {
        return api.post('/server/version/install', { version });
    },
};
```

### 4.5 WebSocket Subscription

The existing websocket hub already supports naming rooms. The version manager uses a new room `"server_version"`:

```javascript
socket.emit('subscribe', 'server_version');
socket.on('server_version', (data) => {
    // data = { percent: 45, status: "Extracting..." }
});
```

On the Go side, progress is broadcast via:

```go
wsRoom := websocket.WebsocketHub.GetRoom("server_version")
wsRoom.Send(progressJSON)
```

---

## 5. Version Comparison Logic

The existing `factorio.Version` type (`version.go`) is a `[4]uint` and already provides:

| Method | Usage |
|---|---|
| `v.Less(b)` | Check if installed < available |
| `v.Greater(b)` | Check if installed > available |
| `v.Equals(b)` | Check equality |
| `v.String()` | Display: "2.0.28" |
| `v.UnmarshalText()` | Parse from string |

Factorio uses 4-part versions: `major.minor.patch.build` (e.g. `2.0.28.0`). The build number is usually 0 for released builds. Comparison logic is already correct for our use case.

**Update available check:**
```go
func IsUpdateAvailable(installed, latestStable Version) bool {
    return latestStable.Greater(installed)
}
```

**Version compatibility filter (future):** The existing `GreaterC` method handles experimental/super-major incompatibility. Not needed for v1 since we only list versions >= 2.0.

---

## 6. Download Progress

### 6.1 WebSocket Event Format

The backend emits progress events to the `"server_version"` websocket room:

```json
{
    "room": "server_version",
    "event": "install_progress",
    "data": {
        "percent": 45,
        "stage": "download",
        "message": "Downloading... 45.2 MB / 102.4 MB"
    }
}
```

Stages:

| stage | percent range | meaning |
|---|---|---|
| `download` | 0 - 90 | Downloading tar.xz from factorio.com |
| `extract` | 90 - 99 | Extracting archive over FactorioDir |
| `verify` | 99 - 100 | Running factorio --version on new binary |
| `done` | 100 | Complete |
| `error` | -- | Installation failed |

### 6.2 Backend Implementation

```go
func reportProgress(percent int, stage, message string) {
    wsRoom := websocket.WebsocketHub.GetRoom("server_version")
    data, _ := json.Marshal(map[string]interface{}{
        "room": "server_version",
        "event": "install_progress",
        "data": map[string]interface{}{
            "percent": percent,
            "stage":   stage,
            "message": message,
        },
    })
    wsRoom.Send(string(data))
}
```

The HTTP download uses a `io.TeeReader` with a `ProgressReader` wrapper:

```go
type ProgressReader struct {
    reader     io.Reader
    total      int64
    read       int64
    lastReport int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.reader.Read(p)
    pr.read += int64(n)
    newPct := pr.read * 100 / pr.total
    lastPct := pr.lastReport * 100 / pr.total
    if newPct > lastPct+4 || newPct >= 100 {
        pr.lastReport = pr.read
        reportProgress(int(newPct), "download",
            fmt.Sprintf("%.1f MB / %.1f MB",
                float64(pr.read)/1024/1024,
                float64(pr.total)/1024/1024))
    }
    return n, err
}
```

### 6.3 Frontend Progress Bar

A Tailwind-based progress bar inside the Panel when `status === 'installing'`:

```jsx
{isInstalling && progress && (
    <div className="w-full bg-gray-light rounded h-4 mt-2">
        <div className="bg-orange h-4 rounded transition-all duration-300"
             style={{ width: `${progress.percent}%` }} />
        <span className="text-sm text-dirty-white ml-2">
            {progress.message}
        </span>
    </div>
)}
```

---

## 7. Error Handling

### 7.1 Error Scenarios

| Scenario | Detection | Response |
|---|---|---|
| Server running | ServerOffMiddleware in routes.go | 423 Locked + message |
| No credentials saved | credentials.Load() returns false | 403 + "Log in via Mod Portal" |
| Network failure | http.Get timeout / DNS error | 502 + "Download failed: {details}" |
| Invalid version string | version.UnmarshalText() fails | 400 + "Invalid version" |
| Disk full / permission denied | os.Create / io.Copy fails | 500 + "Install failed: disk error" |
| Corrupt download | xz decompression fails | 500 + "Download corrupt, try again" |
| Binary not found after install | exec.Command fails | 500 + "New binary could not be verified" |
| Extract permission denied | tar.Reader write fails | 500 + "Permission denied extracting" |

### 7.2 Atomicity

The install operation is NOT fully atomic (replacing the binary while the server is stopped is inherently safe). However:

- The temp download file is cleaned up in a `defer` block even on error.
- If extraction fails partway, the FactorioDir is left in an inconsistent state. To mitigate, v1 documents this risk. v2 could add a backup step:

```go
// v1: direct overwrite
// v2: backup bin/x64/factorio before extracting, restore on failure
```

### 7.3 Frontend Error Display

```jsx
{error && (
    <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
        <span className="block sm:inline">{error}</span>
    </div>
)}
```

---

## 8. Integration with Existing Auth

### 8.1 Token Source

The Factorio download API requires `?username=<user>&token=<token>` query parameters on the download URL. This is the same authentication used for mod downloads in `mod_Mods.go`.

The credentials are stored in `./factorio.auth` (two-line JSON: `{"username": "...", "userkey": "..."}`), managed by the existing `credentials.go` package.

### 8.2 Auth Flow

```
1. User configures token via Mod Portal settings page (existing feature)
   POST /api/mods/portal/login { username, token }
   factorio.auth file saved

2. Version install API reads the same credentials
   credentials.Load() in version_manager.go

3. Download URL constructed:
   https://factorio.com/get-download/{version}/headless/linux64?username={u}&token={t}

4. If credentials missing, 403 error returned to UI
   UI shows link to Mod Portal settings page
```

### 8.3 Credential Validation

The Factorio download API returns 403 if the token is invalid or the user does not own the game. This is surfaced as:

```json
{
    "error": "Factorio.com authentication failed. Check your Mod Portal token."
}
```

---

## 9. Sidebar Integration

### 9.1 Layout Changes (`ui/App/components/Layout.jsx`)

Add a new link inside the existing "Server Management" section, before the closing `</div>` and after `linkLogs`:

```jsx
<Link to="/server-version">{t('linkServerVersion', { ns: 'layout' })}</Link>
```

Place it after the `linkLogs` link to group it with server management operations.

### 9.2 Layout i18n Keys

Add to `ui/locales/en/layout.json`:
```json
"linkServerVersion": "Server Version"
```

Add to `ui/locales/zh-CN/layout.json`:
```json
"linkServerVersion": "服务器版本"
```

### 9.3 Route Registration (`ui/App/App.jsx`)

```jsx
import ServerVersion from "./views/ServerVersion";

// Inside the ProtectedRoute > Layout route group:
<Route path="server-version" element={<ServerVersion serverStatus={serverStatus} />} />
```

### 9.4 Frontend Route Registration (Go backend)

Add to `src/api/routes.go` subrouter path routing:

```go
subRouter.Path("/server-version").
    Methods("GET").
    Name("Server Version").
    Handler(http.StripPrefix("/server-version", http.FileServer(http.Dir("./app/"))))
```

---

## 10. i18n

### 10.1 New Namespace: `serverVersion`

Create two new files:

**`ui/locales/en/serverVersion.json`**:
```json
{
    "title": "Server Version",
    "installedVersion": "Installed Version",
    "current": "Current",
    "unknown": "Unknown",
    "upToDate": "Up to date",
    "updateAvailable": "Update Available",
    "updateTo": "Update to {{version}}",
    "availableVersions": "Available Versions",
    "stable": "stable",
    "experimental": "experimental",
    "latestStable": "latest stable",
    "install": "Install",
    "installing": "Installing...",
    "confirmInstall": "Are you sure you want to install Factorio {{version}}? The server must be stopped.",
    "installSuccess": "Successfully installed Factorio {{version}}",
    "installFailed": "Installation failed: {{error}}",
    "loginRequired": "Factorio.com login required.",
    "loginLink": "Go to Mod Portal settings",
    "previousVersion": "Previous version",
    "downloadStage": "Downloading...",
    "extractStage": "Extracting...",
    "verifyStage": "Verifying..."
}
```

**`ui/locales/zh-CN/serverVersion.json`**:
```json
{
    "title": "服务器版本",
    "installedVersion": "已安装版本",
    "current": "当前版本",
    "unknown": "未知",
    "upToDate": "已是最新",
    "updateAvailable": "有可用更新",
    "updateTo": "更新至 {{version}}",
    "availableVersions": "可用版本",
    "stable": "稳定版",
    "experimental": "实验版",
    "latestStable": "最新稳定版",
    "install": "安装",
    "installing": "安装中...",
    "confirmInstall": "确定要安装 Factorio {{version}} 吗？服务器必须处于停止状态。",
    "installSuccess": "Factorio {{version}} 安装成功",
    "installFailed": "安装失败: {{error}}",
    "loginRequired": "需要登录 Factorio.com。",
    "loginLink": "前往 Mod Portal 设置",
    "previousVersion": "上一个版本",
    "downloadStage": "下载中...",
    "extractStage": "解压中...",
    "verifyStage": "验证中..."
}
```

### 10.2 i18n Registration

Add to `ui/i18n.js`:

```js
import enServerVersion from './locales/en/serverVersion.json';
import zhServerVersion from './locales/zh-CN/serverVersion.json';

// In resources object under each language:
serverVersion: enServerVersion,
serverVersion: zhServerVersion,
```

---

## 11. Implementation Order

### Phase 1: Backend Core

| Step | File | What |
|---|---|---|
| 1.1 | `src/factorio/version_manager.go` | Add `GetInstalledVersion()`, `FetchAvailableVersions()`, `DownloadAndInstall()`, `extractTarXz()` |
| 1.2 | `src/api/version_handler.go` | Add `CurrentVersionHandler`, `AvailableVersionsHandler`, `InstallVersionHandler` |
| 1.3 | `src/api/routes.go` | Register three new routes in `apiRoutes` |
| 1.4 | `src/go.mod` | Add `github.com/ulikunitz/xz` dependency |

### Phase 2: Frontend

| Step | File | What |
|---|---|---|
| 2.1 | `ui/api/resources/server.js` | Add `installVersion()` method |
| 2.2 | `ui/App/views/ServerVersion.jsx` | Create new view component |
| 2.3 | `ui/App/App.jsx` | Add route for `/server-version` |
| 2.4 | `ui/App/components/Layout.jsx` | Add sidebar link |
| 2.5 | `src/api/routes.go` | Add frontend route for `/server-version` |
| 2.6 | `ui/locales/en/serverVersion.json` | English translations |
| 2.7 | `ui/locales/zh-CN/serverVersion.json` | Chinese translations |
| 2.8 | `ui/i18n.js` | Register new namespace |

### Phase 3: Polish and Testing

| Step | What |
|---|---|
| 3.1 | Verify progress events reach frontend |
| 3.2 | Test download failure / resume |
| 3.3 | Test auth credential expiry |
| 3.4 | Error message review (all 7 error scenarios) |
| 3.5 | Verify binary still runs after install |

---

## 12. Testing Criteria

### 12.1 Backend Tests (Go)

| Test | Input | Expected |
|---|---|---|
| `TestGetInstalledVersion` | Existing factorio binary at configured path | Valid `Version` parsed from `--version` output |
| `TestFetchAvailableVersions` | Mock HTTP server returning `latest-releases` JSON | `[]AvailableVersion` with stable + experimental |
| `TestDownloadAndInstall` | Mock HTTP server serving a valid tar.xz | Binary at FactorioBinary path, extracted correctly |
| `TestInstallRequiresCredentials` | No factorio.auth file | Error returned |
| `TestInstallWithBadVersion` | Version "0.0.0" | Invalid version error |
| `TestVersionComparison` | "(2.0.28, 2.0.32)" | `IsUpdateAvailable` returns `true` |
| `TestVersionComparison_Equal` | "(2.0.32, 2.0.32)" | `IsUpdateAvailable` returns `false` |

### 12.2 Frontend Tests (Manual)

| Scenario | Steps | Expected |
|---|---|---|
| Load page, server stopped | Navigate to /server-version | Shows current version + available list |
| Update available | Installed is older than stable | Shows "Update Available" + CTA button |
| Install starts | Click "Install" | Shows progress bar, buttons disabled |
| Install succeeds | Wait for completion | Shows new version, "Up to date" status |
| Install fails (network) | Simulate offline | Error message displayed, buttons re-enabled |
| Install fails (no auth) | Delete factorio.auth | 403 error, link to Mod Portal |
| Server running | Start server, try to install | 423 error, "Stop server first" message |
| Progress updates | Watch websocket during install | Progress bar fills smoothly |

### 12.3 Integration Test

```
1. Start FSM with known Factorio installation (e.g. 2.0.28)
2. Navigate to Server Version page
3. Verify "Current: 2.0.28" displayed
4. Click "Install" on 2.0.32
5. Wait for progress bar to reach 100%
6. Verify page now shows "Current: 2.0.32"
7. Start server, verify it runs on new version
```

---

## Appendix A: Factorio Download API Reference

**List latest releases (no auth):**
```
GET https://factorio.com/api/latest-releases
```
Response:
```json
{
    "stable": {
        "alpha": "2.0.32",
        "demo": "2.0.32",
        "headless": "2.0.32"
    },
    "experimental": {
        "alpha": "2.0.32",
        "demo": "2.0.32",
        "headless": "2.0.32"
    }
}
```

**Download headless (requires auth):**
```
GET https://factorio.com/get-download/{version}/headless/linux64?username={user}&token={token}
```
Returns `application/x-xz` binary stream.

## Appendix B: Files Changed Summary

| File | Change Type |
|---|---|
| `src/factorio/version_manager.go` | **New** |
| `src/api/version_handler.go` | **New** |
| `src/api/routes.go` | Modify (add 3 routes) |
| `src/go.mod` | Modify (add `ulikunitz/xz`) |
| `src/go.sum` | Modify (auto) |
| `ui/App/views/ServerVersion.jsx` | **New** |
| `ui/App/App.jsx` | Modify (add import + route) |
| `ui/App/components/Layout.jsx` | Modify (add sidebar link) |
| `ui/api/resources/server.js` | Modify (add `installVersion`) |
| `ui/locales/en/serverVersion.json` | **New** |
| `ui/locales/zh-CN/serverVersion.json` | **New** |
| `ui/i18n.js` | Modify (register new namespace) |

---

## Appendix C: Open Questions for Implementation

1. Should the version range (for the `available` endpoint) be hardcoded as `[2.0.0 ... latest]` or fetched from a community index? v1 uses a hardcoded range with step increments. If the Factorio API later provides a version list endpoint, switch to that.

2. What is the minimum Factorio version this should support? v1 targets >= 2.0.0. Pre-2.0 versions are out of support.

3. Should we preserve the old binary as `factorio.bak` before overwriting? v1 says no (clean override). v2 can add this.
