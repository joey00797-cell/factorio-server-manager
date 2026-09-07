package api

import (
	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"net/http"

	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
	ServerOff   bool // Set to `true' if factorio server has to be turned off to call this
}

type Routes []Route

func ServerOffMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only run if server is turned off
		server, ok := serverFromRequest(r)
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		if server.GetRunning() {
			http.Error(w, "factorio server still running", http.StatusLocked)
		} else {
			next.ServeHTTP(w, r)
		}
		return
	})
}

func NewRouter() *mux.Router {
	mainRouter := mux.NewRouter().StrictSlash(true)

	// create subrouter for authenticated calls
	subRouter := mainRouter.NewRoute().Subrouter()
	subRouter.Use(AuthMiddleware)

	mainRouter.Path("/api/locales/list").
		Methods("GET").
		Name("PublicLocalesList").
		HandlerFunc(LocalesListHandler)
	mainRouter.Path("/api/locales/template").
		Methods("GET").
		Name("PublicLocalesTemplate").
		HandlerFunc(LocalesTemplateHandler)
	mainRouter.Path("/api/locales/{lang}").
		Methods("GET").
		Name("PublicLocalesGet").
		HandlerFunc(LocalesGetHandler)

	mainRouter.Path("/api/info").
		Methods("GET").
		Name("Info").
		HandlerFunc(InfoHandler)

	// API subrouter
	// Serves all JSON REST handlers prefixed with /api
	apiRouter := mainRouter.PathPrefix("/api").Subrouter()
	apiRouter.Use(AuthMiddleware)

	// use subrouter for calls, that run only, when server is turned off
	serverOffRouter := apiRouter.NewRoute().Subrouter()
	serverOffRouter.Use(ServerOffMiddleware)

	apiRouter.NewRoute().Subrouter()
	for _, route := range apiRoutes {
		var router *mux.Router
		if route.ServerOff {
			router = serverOffRouter
		} else {
			router = apiRouter
		}
		router.Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

	// The login handler does not check for authentication.
	mainRouter.Path("/api/login").
		Methods("POST").
		Name("LoginUser").
		HandlerFunc(LoginUser)

	// Public modpack download — no auth required, safe URL without /api/ prefix
	mainRouter.Path("/share/{modpack}").
		Methods("GET").
		Name("PublicModPackDownload").
		HandlerFunc(PublicModPackDownloadHandler)

	// Route for initializing websocket connection
	// Clients connecting to /ws establish websocket connection by upgrading
	// HTTP session.
	// Ensure user is logged in with the AuthorizeHandler middleware
	subRouter.Path("/ws").
		Methods("GET").
		Name("Websocket").
		Handler(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					websocket.ServeWs(w, r)
				},
			),
		)

	// Serves the frontend application from the app directory
	mainRouter.Path("/favicon.ico").
		Methods("GET").
		Name("Favicon").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/favicon.ico")
		})

	// Uses basic file server to serve index.html and Javascript application
	// Routes match the ones defined in React frontend application
	mainRouter.Path("/login").
		Methods("GET").
		Name("Login").
		Handler(http.StripPrefix("/login", http.FileServer(http.Dir("./app/"))))

	subRouter.Path("/saves").
		Methods("GET").
		Name("Saves").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/mods").
		Methods("GET").
		Name("Mods").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/server-settings").
		Methods("GET").
		Name("Server settings").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/game-settings").
		Methods("GET").
		Name("Game settings").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/console").
		Methods("GET").
		Name("Console").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/logs").
		Methods("GET").
		Name("Logs").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/fsm-logs").
		Methods("GET").
		Name("FSM Logs").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/user-management").
		Methods("GET").
		Name("User management").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.Path("/help").
		Methods("GET").
		Name("Help").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})
	subRouter.PathPrefix("/servers").
		Methods("GET").
		Name("Servers").
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./app/index.html")
		})

	// catch all route
	mainRouter.PathPrefix("/").
		Methods("GET").
		Name("Index").
		Handler(http.FileServer(http.Dir("./app/")))

	return mainRouter
}

// Defines all API REST endpoints
// All routes are prefixed with /api
var apiRoutes = Routes{
	{
		"ListServers",
		"GET",
		"/servers",
		ListServersHandler,
		false,
	}, {
		"CreateServer",
		"POST",
		"/servers",
		CreateServerHandler,
		false,
	}, {
		"PreviewNextServer",
		"GET",
		"/servers/next",
		PreviewNextServerHandler,
		false,
	}, {
		"GetServer",
		"GET",
		"/servers/{serverID}",
		GetServerHandler,
		false,
	}, {
		"UpdateServer",
		"PATCH",
		"/servers/{serverID}",
		UpdateServerHandler,
		false,
	}, {
		"DeleteServer",
		"DELETE",
		"/servers/{serverID}",
		DeleteServerHandler,
		false,
	}, {
		"ScopedServerStatus",
		"GET",
		"/servers/{serverID}/status",
		GetServerHandler,
		false,
	}, {
		"ScopedFactorioVersion",
		"GET",
		"/servers/{serverID}/facVersion",
		FactorioVersion,
		false,
	}, {
		"ScopedStartServer",
		"POST",
		"/servers/{serverID}/start",
		StartServer,
		true,
	}, {
		"ScopedStopServer",
		"POST",
		"/servers/{serverID}/stop",
		StopServer,
		false,
	}, {
		"ScopedKillServer",
		"POST",
		"/servers/{serverID}/kill",
		KillServer,
		false,
	}, {
		"ScopedSaveServer",
		"POST",
		"/servers/{serverID}/save",
		SaveServerHandler,
		false,
	}, {
		"ScopedListSaves",
		"GET",
		"/servers/{serverID}/saves/list",
		ListSaves,
		false,
	}, {
		"ScopedDlSave",
		"GET",
		"/servers/{serverID}/saves/dl/{save}",
		DLSave,
		false,
	}, {
		"ScopedUploadSave",
		"POST",
		"/servers/{serverID}/saves/upload",
		UploadSave,
		false,
	}, {
		"ScopedRemoveSave",
		"GET",
		"/servers/{serverID}/saves/rm/{save}",
		RemoveSave,
		false,
	}, {
		"GetSaveBackupSchedule",
		"GET",
		"/servers/{serverID}/saves/backups/schedule",
		GetSaveBackupScheduleHandler,
		false,
	}, {
		"UpdateSaveBackupSchedule",
		"PUT",
		"/servers/{serverID}/saves/backups/schedule",
		UpdateSaveBackupScheduleHandler,
		false,
	}, {
		"RunSaveBackup",
		"POST",
		"/servers/{serverID}/saves/backups/run",
		RunSaveBackupHandler,
		false,
	}, {
		"ListSaveBackups",
		"GET",
		"/servers/{serverID}/saves/backups",
		ListSaveBackupsHandler,
		false,
	}, {
		"ListAllSaveBackups",
		"GET",
		"/servers/{serverID}/saves/backups/all",
		ListAllSaveBackupsHandler,
		false,
	}, {
		"DeleteSaveBackup",
		"DELETE",
		"/servers/{serverID}/saves/backups/{name}",
		DeleteSaveBackupHandler,
		false,
	}, {
		"DownloadSaveBackup",
		"GET",
		"/servers/{serverID}/saves/backups/{name}/download",
		DownloadSaveBackupHandler,
		false,
	}, {
		"RestoreSaveBackup",
		"POST",
		"/servers/{serverID}/saves/backups/{name}/restore",
		RestoreSaveBackupHandler,
		false,
	}, {
		"TogglePinBackup",
		"POST",
		"/servers/{serverID}/saves/backups/{name}/pin",
		TogglePinBackupHandler,
		false,
	}, {
		"RenameBackup",
		"PUT",
		"/servers/{serverID}/saves/backups/{name}/rename",
		RenameBackupHandler,
		false,
	}, {
		"ScopedCreateSave",
		"GET",
		"/servers/{serverID}/saves/create/{save}",
		CreateSaveHandler,
		true,
	}, {
		"ScopedLogTail",
		"GET",
		"/servers/{serverID}/log/tail",
		LogTail,
		false,
	}, {
		"ScopedLoadConfig",
		"GET",
		"/servers/{serverID}/config",
		LoadConfig,
		false,
	}, {
		"ScopedUpdateConfig",
		"POST",
		"/servers/{serverID}/config/update",
		UpdateConfig,
		false,
	}, {
		"ScopedGetServerSettings",
		"GET",
		"/servers/{serverID}/settings",
		GetServerSettings,
		false,
	}, {
		"ScopedUpdateServerSettings",
		"POST",
		"/servers/{serverID}/settings/update",
		UpdateServerSettings,
		false,
	}, {
		"ModLibraryList",
		"GET",
		"/mods/library",
		ListModAssetsHandler,
		false,
	}, {
		"ModLibraryUpload",
		"POST",
		"/mods/library/upload",
		UploadModToLibraryHandler,
		false,
	}, {
		"ModLibraryPortalImport",
		"POST",
		"/mods/library/portal-import",
		ImportPortalModToLibraryHandler,
		false,
	}, {
		"ModLibraryDelete",
		"DELETE",
		"/mods/library/{id}",
		DeleteModAssetHandler,
		false,
	}, {
		"ScopedGetManifest",
		"GET",
		"/servers/{serverID}/mods/manifest",
		GetManifestHandler,
		false,
	}, {
		"ScopedUpdateManifest",
		"PUT",
		"/servers/{serverID}/mods/manifest",
		UpdateManifestHandler,
		false,
	}, {
		"ScopedPreviewManifest",
		"GET",
		"/servers/{serverID}/mods/manifest/preview",
		PreviewManifestHandler,
		false,
	}, {
		"ScopedApplyManifest",
		"POST",
		"/servers/{serverID}/mods/manifest/apply",
		ApplyManifestHandler,
		false,
	}, {
		"ManifestReset",
		"POST",
		"/servers/{serverID}/mods/manifest/reset",
		ResetManifestHandler,
		false,
	}, {
		"ScopedListInstalledMods",
		"GET",
		"/servers/{serverID}/mods/list",
		ListInstalledModsHandler,
		false,
	}, {
		"ScopedToggleMod",
		"POST",
		"/servers/{serverID}/mods/toggle",
		ModToggleHandler,
		true,
	}, {
		"ScopedDeleteMod",
		"POST",
		"/servers/{serverID}/mods/delete",
		ModDeleteHandler,
		true,
	}, {
		"ScopedDeleteAllMods",
		"POST",
		"/servers/{serverID}/mods/delete/all",
		ModDeleteAllHandler,
		true,
	}, {
		"ScopedUpdateMod",
		"POST",
		"/servers/{serverID}/mods/update",
		ModUpdateHandler,
		true,
	}, {
		"ScopedUploadMod",
		"POST",
		"/servers/{serverID}/mods/upload",
		ModUploadHandler,
		true,
	}, {
		"ScopedDownloadMods",
		"GET",
		"/servers/{serverID}/mods/download",
		ModDownloadHandler,
		false,
	}, {
		"ScopedModPortalInstallMod",
		"POST",
		"/servers/{serverID}/mods/portal/install",
		ModPortalInstallHandler,
		true,
	}, {
		"ScopedModPortalInstallMultiple",
		"POST",
		"/servers/{serverID}/mods/portal/install/multiple",
		ModPortalInstallMultipleHandler,
		true,
	}, {
		"ScopedGetModsFromSave",
		"POST",
		"/servers/{serverID}/saves/mods/list",
		GetModsFromSaveHandler,
		true,
	}, {
		"ScopedSyncModsFromSave",
		"POST",
		"/servers/{serverID}/saves/mods/sync",
		SyncModsFromSaveHandler,
		true,
	}, {
		"ScopedSyncDependencies",
		"POST",
		"/servers/{serverID}/mods/manifest/sync-deps",
		SyncDependenciesHandler,
		false,
	}, {
		"ListPresets",
		"GET",
		"/servers/{serverID}/presets",
		ListPresetsHandler,
		false,
	}, {
		"GetPreset",
		"GET",
		"/servers/{serverID}/presets/{preset}",
		GetPresetHandler,
		false,
	}, {
		"SavePreset",
		"POST",
		"/servers/{serverID}/presets",
		SavePresetHandler,
		false,
	}, {
		"LoadPreset",
		"POST",
		"/servers/{serverID}/presets/{preset}/load",
		LoadPresetHandler,
		false,
	}, {
		"DeletePreset",
		"DELETE",
		"/servers/{serverID}/presets/{preset}",
		DeletePresetHandler,
		false,
	}, {
		"AddModToPreset",
		"POST",
		"/servers/{serverID}/presets/{preset}/mods",
		AddModToPresetHandler,
		false,
	}, {
		"RemoveModFromPreset",
		"DELETE",
		"/servers/{serverID}/presets/{preset}/mods",
		RemoveModFromPresetHandler,
		false,
	}, {
		"ScopedModSettings",
		"GET",
		"/servers/{serverID}/mod-settings",
		GetModSettingsHandler,
		false,
	}, {
		"ScopedUpdateModSettings",
		"POST",
		"/servers/{serverID}/mod-settings",
		UpdateModSettingsHandler,
		false,
	},
	{
		"ListSaves",
		"GET",
		"/saves/list",
		ListSaves,
		false,
	}, {
		"DlSave",
		"GET",
		"/saves/dl/{save}",
		DLSave,
		false,
	}, {
		"UploadSave",
		"POST",
		"/saves/upload",
		UploadSave,
		false,
	}, {
		"RemoveSave",
		"GET",
		"/saves/rm/{save}",
		RemoveSave,
		false,
	}, {
		"CreateSave",
		"GET",
		"/saves/create/{save}",
		CreateSaveHandler,
		true,
	}, {
		"LoadModsFromSave",
		"POST",
		"/saves/mods",
		LoadModsFromSaveHandler,
		true,
	}, {
		"GetModsFromSave",
		"POST",
		"/saves/mods/list",
		GetModsFromSaveHandler,
		true,
	}, {
		"SyncModsFromSave",
		"POST",
		"/saves/mods/sync",
		SyncModsFromSaveHandler,
		true,
	}, {
		"CancelSync",
		"POST",
		"/saves/mods/sync/cancel",
		CancelSyncHandler,
		false,
	}, {
		"LogTail",
		"GET",
		"/log/tail",
		LogTail,
		false,
	}, {
		"FSMLogTail",
		"GET",
		"/fsm/log/tail",
		FSMLogTail,
		false,
	}, {
		"LoadConfig",
		"GET",
		"/config",
		LoadConfig,
		false,
	}, {
		"UpdateConfig",
		"POST",
		"/config/update",
		UpdateConfig,
		false,
	}, {
		"StartServer",
		"POST",
		"/server/start",
		StartServer,
		true,
	}, {
		"StopServer",
		"GET",
		"/server/stop",
		StopServer,
		false,
	}, {
		"KillServer",
		"GET",
		"/server/kill",
		KillServer,
		false,
	}, {
		"RunningServer",
		"GET",
		"/server/status",
		CheckServer,
		false,
	}, {
		"FactorioVersion",
		"GET",
		"/server/facVersion",
		FactorioVersion,
		false,
	}, {
		"AvailableVersions",
		"GET",
		"/server/availableVersions",
		AvailableVersions,
		false,
	}, {
		"AvailableVersionsScoped",
		"GET",
		"/versions",
		AvailableVersions,
		false,
	}, {
		"InstalledVersions",
		"GET",
		"/versions/installed",
		InstalledVersionsHandler,
		false,
	}, {
		"DownloadedVersions",
		"GET",
		"/versions/downloaded",
		DownloadedVersionsHandler,
		false,
	}, {
		"DeleteDownload",
		"DELETE",
		"/versions/downloaded/{version}",
		DeleteDownloadHandler,
		false,
	}, {
		"DeleteInstalledVersion",
		"DELETE",
		"/versions/installed/{version}",
		DeleteInstalledVersionHandler,
		false,
	}, {
		"InstallVersion",
		"POST",
		"/versions/install",
		InstallFactorio,
		false,
	}, {
		"InstallFactorio",
		"POST",
		"/server/install",
		InstallFactorio,
		true,
	}, {
		"RemoveFactorio",
		"DELETE",
		"/server/install",
		RemoveFactorio,
		true,
	}, {
		"GetInstallStatus",
		"GET",
		"/server/install/status",
		GetInstallStatus,
		false,
	}, {
		"LogoutUser",
		"GET",
		"/logout",
		LogoutUser,
		false,
	}, {
		"StatusUser",
		"GET",
		"/user/status",
		GetCurrentLogin,
		false,
	}, {
		"ListUsers",
		"GET",
		"/user/list",
		ListUsers,
		false,
	}, {
		"AddUser",
		"POST",
		"/user/add",
		AddUser,
		false,
	}, {
		"RemoveUser",
		"POST",
		"/user/remove",
		RemoveUser,
		false,
	}, {
		"ChangePassword",
		"POST",
		"/user/password",
		ChangePassword,
		false,
	}, {
		"GetServerSettings",
		"GET",
		"/settings",
		GetServerSettings,
		false,
	}, {
		"UpdateServerSettings",
		"POST",
		"/settings/update",
		UpdateServerSettings,
		false,
	},
	// Mod Portal Stuff
	{
		"ModPortalListAllMods",
		"GET",
		"/mods/portal/list",
		ModPortalListModsHandler,
		false,
	}, {
		"ModPortalGetModInfo",
		"GET",
		"/mods/portal/info/{mod}",
		ModPortalModInfoHandler,
		false,
	}, {
		"ModPortalInstallMod",
		"POST",
		"/mods/portal/install",
		ModPortalInstallHandler,
		true,
	}, {
		"ModPortalLogin",
		"POST",
		"/mods/portal/login",
		ModPortalLoginHandler,
		false,
	}, {
		"ModPortalLoginStatus",
		"GET",
		"/mods/portal/loginstatus",
		ModPortalLoginStatusHandler,
		false,
	}, {
		"ModPortalLogout",
		"GET",
		"/mods/portal/logout",
		ModPortalLogoutHandler,
		false,
	}, {
		"ModPortalInstallMultiple",
		"POST",
		"/mods/portal/install/multiple",
		ModPortalInstallMultipleHandler,
		true,
	},
	// Mods Stuff
	{
		"ListInstalledMods",
		"GET",
		"/mods/list",
		ListInstalledModsHandler,
		false,
	}, {
		"ToggleMod",
		"POST",
		"/mods/toggle",
		ModToggleHandler,
		true,
	}, {
		"DeleteMod",
		"POST",
		"/mods/delete",
		ModDeleteHandler,
		true,
	}, {
		"DeleteAllMods",
		"POST",
		"/mods/delete/all",
		ModDeleteAllHandler,
		true,
	}, {
		"UpdateMod",
		"POST",
		"/mods/update",
		ModUpdateHandler,
		true,
	}, {
		"UploadMod",
		"POST",
		"/mods/upload",
		ModUploadHandler,
		true,
	}, {
		"DownloadMods",
		"GET",
		"/mods/download",
		ModDownloadHandler,
		false,
	},
	// Mod Packs
	{
		"ModPacksList",
		"GET",
		"/mods/packs/list",
		ModPackListHandler,
		false,
	}, {
		"ModPackCreate",
		"POST",
		"/mods/packs/create",
		ModPackCreateHandler,
		false,
	}, {
		"ModPackDelete",
		"POST",
		"/mods/packs/{modpack}/delete",
		ModPackDeleteHandler,
		false,
	}, {
		"ModPackDownload",
		"GET",
		"/mods/packs/{modpack}/download",
		ModPackDownloadHandler,
		false,
	}, {
		"LoadModPack",
		"POST",
		"/mods/packs/{modpack}/load",
		ModPackLoadHandler,
		true,
	},
	// Mods inside Mod Packs
	{
		"ModPackListMods",
		"GET",
		"/mods/packs/{modpack}/list",
		ModPackModListHandler,
		false,
	}, {
		"ModPackToggleMod",
		"POST",
		"/mods/packs/{modpack}/mod/toggle",
		ModPackModToggleHandler,
		false,
	}, {
		"ModPackDeleteMod",
		"POST",
		"/mods/packs/{modpack}/mod/delete",
		ModPackModDeleteHandler,
		false,
	}, {
		"ModPackDeleteAllMod",
		"POST",
		"/mods/packs/{modpack}/mod/delete/all",
		ModPackModDeleteAllHandler,
		false,
	}, {
		"ModPackUpdateMod",
		"POST",
		"/mods/packs/{modpack}/mod/update",
		ModPackModUpdateHandler,
		false,
	}, {
		"ModPackUploadMod",
		"POST",
		"/mods/packs/{modpack}/mod/upload",
		ModPackModUploadHandler,
		false,
	}, {
		"ModPackModPortalInstallMod",
		"POST",
		"/mods/packs/{modpack}/portal/install",
		ModPackModPortalInstallHandler,
		false,
	}, {
		"ModPackModPortalInstallMultiple",
		"POST",
		"/mods/packs/{modpack}/portal/install/multiple",
		ModPackModPortalInstallMultipleHandler,
		false,
	}, {
		"GetAutostart",
		"GET",
		"/autostart",
		GetAutostartHandler,
		false,
	}, {
		"SetAutostart",
		"POST",
		"/autostart",
		SetAutostartHandler,
		false,
	}, {
		"LocalesList",
		"GET",
		"/locales/list",
		LocalesListHandler,
		false,
	}, {
		"LocalesTemplate",
		"GET",
		"/locales/template",
		LocalesTemplateHandler,
		false,
	}, {
		"LocalesGet",
		"GET",
		"/locales/{lang}",
		LocalesGetHandler,
		false,
	}, {
		"LocalesUpload",
		"POST",
		"/locales/upload",
		LocalesUploadHandler,
		false,
	},
}
