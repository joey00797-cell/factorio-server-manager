package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/sessions"

	"github.com/gorilla/mux"
)

const readHttpBodyError = "Could not read the Request Body."

type JSONResponseFileInput struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,string"`
	Error     string      `json:"error"`
	ErrorKeys []int       `json:"errorkeys"`
}

func WriteResponse(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error writing response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ReadRequestBody(w http.ResponseWriter, r *http.Request) (body []byte, resp interface{}, err error) {
	if r.Body == nil {
		resp = fmt.Sprintf("%s: no request body", readHttpBodyError)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		err = errors.New("no request body")
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		resp = fmt.Sprintf("%s: %s", readHttpBodyError, err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
	}
	return
}

func ReadSessionStore(w http.ResponseWriter, r *http.Request, name string) (session *sessions.Session, resp interface{}, err error) {
	session, err = sessionStore.Get(r, name)
	if err != nil {
		resp = fmt.Sprintf("Error reading session cookie [%s]: %s", name, err)
		log.Println(resp)
		if session != nil {
			session.Options.MaxAge = -1
			err2 := session.Save(r, w)
			if err2 != nil {
				log.Printf("Error deleting session cookie: %s", err2)
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	}
	return
}

func SaveSession(w http.ResponseWriter, r *http.Request, session *sessions.Session) (resp interface{}, err error) {
	err = session.Save(r, w)
	if err != nil {
		resp = fmt.Sprintf("Error saving session cookie: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
	}
	return
}

// Lists all save files in the factorio/saves directory
func ListSaves(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	latestParam := r.URL.Query().Get("latest")

	var withLatest bool

	if latestParam != "" {
		var err error
		withLatest, err = strconv.ParseBool(latestParam)
		if err != nil {
			resp = fmt.Sprintf("Error parsing latestParam: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	savesList, err := factorio.ListSaves()
	if err != nil {
		resp = fmt.Sprintf("Error listing save files: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get actual latest and add name
	// but only if requested
	if withLatest && len(savesList) != 0 {
		latestSave, err := factorio.GetLatestSave()
		if err != nil {
			resp = fmt.Sprintf("Error getting latest save: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		latestSave.Name = fmt.Sprintf("Load Latest (%s)", latestSave.Name)
		savesList = append(savesList, latestSave)
	}

	resp = savesList
}

func DLSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	config := bootstrap.GetConfig()
	vars := mux.Vars(r)
	save := vars["save"]
	saveName := filepath.Join(config.FactorioSavesDir, save)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", save))
	log.Printf("%s downloading: %s", r.Host, saveName)

	http.ServeFile(w, r, saveName)
}

func UploadSave(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	log.Println("Uploading save file")

	r.ParseMultipartForm(32 << 20)
	config := bootstrap.GetConfig()

	for _, saveFile := range r.MultipartForm.File["savefile"] {
		ext := filepath.Ext(saveFile.Filename)
		if ext != ".zip" {
			// Only zip-files allowed
			resp = fmt.Sprintf("Fileformat {%s} is not allowed", ext)
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		file, err := saveFile.Open()
		if err != nil {
			resp = fmt.Sprintf("Error opening uploaded saveFile: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer file.Close()

		out, err := os.Create(filepath.Join(config.FactorioSavesDir, saveFile.Filename))
		if err != nil {
			resp = fmt.Sprintf("Error creating new savefile to copy uploaded on to: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			resp = fmt.Sprintf("Error coping uploaded file to created file on disk: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	resp = "Uploading files successful"
}

// Deletes provided save
func RemoveSave(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	name := vars["save"]

	save, err := factorio.FindSave(name)
	if err != nil {
		resp = fmt.Sprintf("Error finding save {%s}: %s", name, err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = save.Remove()
	if err != nil {
		resp = fmt.Sprintf("Error removing save {%s}: %s", name, err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// save was removed
	resp = fmt.Sprintf("Removed save: %s", save.Name)
}

// Launches Factorio server binary with --create flag to create save
// Url must include save name for creation of savefile
func CreateSaveHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	vars := mux.Vars(r)
	saveName := vars["save"]

	if saveName == "" {
		resp = fmt.Sprintf("Error creating save, no save name provided: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	config := bootstrap.GetConfig()
	saveFile := filepath.Join(config.FactorioSavesDir, saveName)
	cmdOut, err := factorio.CreateSave(saveFile)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			resp = "saves.factorio_not_installed"
		} else {
			resp = fmt.Sprintf("Error creating save {%s}: %s", saveName, err)
		}
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = fmt.Sprintf("Save %s created successfully. Command output: \n%s", saveName, cmdOut)
}

// LogTail returns last lines of the factorio-current.log file
func LogTail(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	config := bootstrap.GetConfig()
	resp, err = factorio.TailLog()
	if err != nil {
		resp = fmt.Sprintf("Could not tail %s: %s", config.FactorioLog, err)
		return
	}
}

// LoadConfig returns JSON response of config.ini file
func LoadConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	config := bootstrap.GetConfig()
	configContents, err := factorio.LoadConfig(config.FactorioConfigFile)
	if err != nil {
		log.Printf("config.ini not available: %s", err)
		resp = map[string]interface{}{}
		return
	}

	resp = configContents

	log.Printf("Sent config.ini response")
}

func StartServer(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}
	var server = factorio.GetFactorioServer()
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	if server.GetRunning() {
		resp = "Factorio server is already running"
		w.WriteHeader(http.StatusConflict)
		return
	}

	log.Printf("Starting Factorio server.")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}

	log.Printf("Starting Factorio server with settings: %v", string(body))

	err = json.Unmarshal(body, &server)
	if err != nil {
		resp = fmt.Sprintf("Error unmarshalling server settings JSON: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if savefile was submitted with request to start server.
	if server.Savefile == "" {
		resp = "Error starting Factorio server: No save file provided"
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go func() {
		err = server.Run()
		if err != nil {
			log.Printf("Error starting Factorio server: %+v", err)
			return
		}
	}()

	timeout := 0
	for timeout <= 3 {
		time.Sleep(1 * time.Second)
		if server.GetRunning() {
			log.Printf("Running Factorio server detected")
			break
		} else {
			log.Printf("Did not detect running Factorio server attempt: %+v", timeout)
		}

		timeout++
	}

	if server.GetRunning() == false {
		resp = fmt.Sprintf("Error starting Factorio server: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = fmt.Sprintf("Factorio server with save: %s started on port: %d", server.Savefile, server.Port)
	log.Println(resp)
}

func StopServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	if server.GetRunning() {
		err := server.Stop()
		if err != nil {
			resp = fmt.Sprintf("Error stopping factorio server: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp = fmt.Sprintf("Factorio server stopped")
		log.Println(resp)
	} else {
		resp = "Factorio server is not running"
		w.WriteHeader(http.StatusConflict)
		return
	}
}

func KillServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	if server.GetRunning() {
		err := server.Kill()
		if err != nil {
			resp = fmt.Sprintf("Error killing factorio server: %s", err)
			log.Println(resp)
			return
		}

		log.Printf("Killed Factorio server.")
		resp = fmt.Sprintf("Factorio server killed")
	} else {
		resp = "Factorio server is not running"
		w.WriteHeader(http.StatusBadRequest)
	}
}

func CheckServer(w http.ResponseWriter, r *http.Request) {
	defer func() {
		WriteResponse(w, factorio.GetFactorioServer())
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
}

func FactorioVersion(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	var server = factorio.GetFactorioServer()
	resp["version"] = server.Version.String()
	resp["base_mod_version"] = server.BaseModVersion
}

// Unmarshall the User object from the given bytearray
// This function has side effects (it will write to resp and to w, in case of an error)
func UnmarshallUserJson(body []byte, w http.ResponseWriter) (user User, resp interface{}, err error) {
	err = json.Unmarshal(body, &user)
	if err != nil {
		resp = fmt.Sprintf("Unable to parse the request body: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
	}
	return
}

// Handler for the Login
func LoginUser(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	// add resp to the response
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}

	user, resp, err := UnmarshallUserJson(body, w)
	if err != nil {
		return
	}

	log.Printf("Logging in user: %s", user.Username)

	err = auth.checkPassword(user.Username, user.Password)
	if err != nil {
		resp = fmt.Sprintf("Password for user %s wrong", user.Username)
		log.Println(resp)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	session, resp, err := ReadSessionStore(w, r, "authentication")
	if err != nil {
		return
	}

	session.Values["username"] = user.Username

	resp, err = SaveSession(w, r, session)
	if err != nil {
		return
	}

	log.Printf("User: %s, logged in successfully", user.Username)

	user.Password = ""
	resp = user
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	session, resp, err := ReadSessionStore(w, r, "authentication")
	if err != nil {
		return
	}

	delete(session.Values, "username")

	resp, err = SaveSession(w, r, session)
	if err != nil {
		return
	}

	resp = "User logged out successfully."
}

func GetCurrentLogin(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	// add resp to the response
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	session, resp, err := ReadSessionStore(w, r, "authentication")
	if err != nil {
		return
	}

	username := session.Values["username"].(string)

	user, err := auth.getUser(username)
	if err != nil {
		resp = fmt.Sprintf("Error getting user: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user.Password = ""

	resp = user
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	users, err := auth.listUsers()
	if err != nil {
		resp = fmt.Sprintf("Error listing users: %s", err)
		log.Println(resp)
		return
	}

	resp = users
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}

	user, resp, err := UnmarshallUserJson(body, w)
	if err != nil {
		return
	}

	err = auth.addUser(user)
	if err != nil {
		resp = fmt.Sprintf("Error in adding user {%s}: %s", user.Username, err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = fmt.Sprintf("User: %s successfully added.", user.Username)
}

func RemoveUser(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}

	user, resp, err := UnmarshallUserJson(body, w)
	if err != nil {
		return
	}

	err = auth.deleteUser(user.Username)
	if err != nil {
		resp = fmt.Sprintf("Error in removing user {%s}, error: %s", user.Username, err)
		log.Println(resp)
		return
	}

	resp = fmt.Sprintf("User: %s successfully removed.", user.Username)
}

func ChangePassword(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}

	var user struct {
		OldPassword        string `json:"old_password"`
		NewPassword        string `json:"new_password"`
		NewPasswordConfirm string `json:"new_password_confirmation"`
	}
	err = json.Unmarshal(body, &user)
	if err != nil {
		resp = fmt.Sprintf("Unable to parse the request body: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// only allow to change its own password
	// get username from session cookie
	session, resp, err := ReadSessionStore(w, r, "authentication")
	if err != nil {
		return
	}

	username := session.Values["username"].(string)

	// check if password for user is correct
	err = auth.checkPassword(username, user.OldPassword)
	if err != nil {
		resp = fmt.Sprintf("Password for user %s wrong", username)
		log.Println(resp)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// only run, when confirmation correct
	if user.NewPassword != user.NewPasswordConfirm {
		resp = fmt.Sprintf("Password confirmation incorrect")
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = auth.changePassword(username, user.NewPassword)
	if err != nil {
		resp = fmt.Sprintf("Error changing password: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = true
}

// GetServerSettings returns JSON response of server-settings.json file
func GetServerSettings(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	config := bootstrap.GetConfig()
	data, err := os.ReadFile(config.SettingsFile)
	if err != nil {
		log.Printf("Error reading server settings file: %v", err)
		resp = map[string]interface{}{}
		return
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil || settings == nil {
		log.Printf("Error parsing server settings: %v", err)
		resp = map[string]interface{}{}
		return
	}
	resp = settings
	log.Printf("Sent server settings response (%d keys)", len(settings))
}

func UpdateServerSettings(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	body, resp, err := ReadRequestBody(w, r)
	if err != nil {
		return
	}
	// Логируем только изменённые поля
	var incomingSettings map[string]interface{}
	if err := json.Unmarshal(body, &incomingSettings); err == nil {
		diffConfig := bootstrap.GetConfig()
		existingRaw, _ := os.ReadFile(diffConfig.SettingsFile)
		var existingForDiff map[string]interface{}
		json.Unmarshal(existingRaw, &existingForDiff)
		for k, v := range incomingSettings {
			if strings.HasPrefix(k, "_comment") {
				continue
			}
			oldV, exists := existingForDiff[k]
			if !exists || fmt.Sprintf("%v", oldV) != fmt.Sprintf("%v", v) {
				log.Printf("Settings changed: %s = %v", k, v)
			}
		}
	}
	config := bootstrap.GetConfig()
	// Читаем текущий файл
	existingData, err2 := os.ReadFile(config.SettingsFile)
	var currentSettings map[string]interface{}
	if err2 == nil {
		json.Unmarshal(existingData, &currentSettings)
	}
	if currentSettings == nil {
		currentSettings = make(map[string]interface{})
	}
	// Мержим новые данные поверх существующих
	var newSettings map[string]interface{}
	if err = json.Unmarshal(body, &newSettings); err != nil {
		resp = fmt.Sprintf("Error unmarshaling settings: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for k, v := range newSettings {
		// не затираем существующее значение если новое = null
		if v != nil {
			currentSettings[k] = v
		}
	}
	var server = factorio.GetFactorioServer()
	server.Settings = currentSettings

	settings, err := json.MarshalIndent(currentSettings, "", "  ")
	if err != nil {
		resp = fmt.Sprintf("Failed to marshal server settings: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = ioutil.WriteFile(config.SettingsFile, settings, 0644)
	if err != nil {
		resp = fmt.Sprintf("Failed to save server settings: %v\n", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("Saved Factorio server settings in server-settings.json")

	if (server.Version.Greater(factorio.Version{0, 17, 0})) {
		// save admins to adminJson
		admins, err := json.MarshalIndent(server.Settings["admins"], "", "  ")
		if err != nil {
			resp = fmt.Sprintf("Failed to marshal admins-Setting: %s", err)
			log.Println(resp)
			return
		}

		err = ioutil.WriteFile(config.FactorioAdminFile, admins, 0664)
		if err != nil {
			resp = fmt.Sprintf("Failed to save admins: %s", err)
			log.Println(resp)
			return
		}
	}

	resp = fmt.Sprintf("Settings successfully saved")
}


// AvailableVersions fetches available Factorio versions from factorio.com API
func AvailableVersions(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json;charset=UTF-8")
    resp, err := http.Get("https://factorio.com/api/latest-releases")
    if err != nil {
        http.Error(w, "Failed to fetch versions", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    w.Write(body)
}

// InstallFactorio downloads and installs a specific Factorio version
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func InstallFactorio(w http.ResponseWriter, r *http.Request) {
    var data struct {
        Version string `json:"version"`
    }
    body, _ := io.ReadAll(r.Body)
    json.Unmarshal(body, &data)

    if data.Version == "" {
        data.Version = "stable"
    }

    status := factorio.GetInstallStatus()
    if status.Installing {
        w.WriteHeader(http.StatusConflict)
        WriteResponse(w, "installation already in progress")
        return
    }

    factorio.SetInstallStatus(factorio.InstallStatus{Installing: true, Version: data.Version, Progress: 0})
    wsRoom := websocket.WebsocketHub.GetRoom("server_version")

    go func() {
    config := bootstrap.GetConfig()
    url := fmt.Sprintf("https://www.factorio.com/get-download/%s/headless/linux64", data.Version)
    log.Printf("[INSTALL] Starting download: %s", url)

    out, err := os.Create("/tmp/factorio_install.tar.xz")
    if err != nil {
        factorio.SetInstallStatus(factorio.InstallStatus{})
        wsRoom.Send(fmt.Sprintf(`{"type":"install_error","error":"%s"}`, err.Error()))
        return
    }
    defer out.Close()

    dlResp, err := http.Get(url)
    if err != nil {
        factorio.SetInstallStatus(factorio.InstallStatus{})
        wsRoom.Send(fmt.Sprintf(`{"type":"install_error","error":"%s"}`, err.Error()))
        return
    }
    defer dlResp.Body.Close()

    pr := &factorio.ProgressReader{
        Reader: dlResp.Body,
        Total:  dlResp.ContentLength,
    }
    pr.OnProgress = func(percent int) {
        factorio.SetInstallStatus(factorio.InstallStatus{Installing: true, Version: data.Version, Progress: percent})
        wsRoom.Send(fmt.Sprintf(`{"type":"download_progress","version":"%s","percent":%d,"current":%d,"total":%d}`, data.Version, percent, pr.Current, pr.Total))
    }
    io.Copy(out, pr)

    extractDir := config.FactorioDir
    log.Printf("[INSTALL] Download complete, extracting to: %s", extractDir)
    wsRoom.Send(fmt.Sprintf(`{"type":"extracting","version":"%s"}`, data.Version))
    os.MkdirAll(extractDir, 0755)
    cmd := exec.Command("tar", "-xf", "/tmp/factorio_install.tar.xz", "-C", extractDir, "--strip-components=1")
    if err := cmd.Run(); err != nil {
        factorio.SetInstallStatus(factorio.InstallStatus{})
        wsRoom.Send(fmt.Sprintf(`{"type":"install_error","error":"extraction failed: %s"}`, err.Error()))
        return
    }
    // Копируем server-settings.json из примера
    settingsDst := config.SettingsFile
    exampleSrc := filepath.Join(extractDir, "data", "server-settings.example.json")
    os.MkdirAll(filepath.Dir(settingsDst), 0755)
    if src, err2 := os.ReadFile(exampleSrc); err2 == nil {
        // Перезаписываем только если файл пустой или содержит null
        existingData, _ := os.ReadFile(settingsDst)
        trimmed := strings.TrimSpace(string(existingData))
        log.Printf("server-settings check: len=%d trimmed=%q", len(trimmed), trimmed[:min(len(trimmed), 20)])
        if trimmed == "" || trimmed == "null" || len(trimmed) < 10 {
            os.WriteFile(settingsDst, src, 0644)
            log.Printf("server-settings.json создан из примера")
            // Перезагружаем настройки в памяти
            srv := factorio.GetFactorioServer()
            if f, err2 := os.Open(settingsDst); err2 == nil {
                json.NewDecoder(f).Decode(&srv.Settings)
                f.Close()
                log.Printf("server-settings.json загружен в память")
            }
        }
    } else {
        log.Printf("Не удалось найти example: %v", err2)
    }

    os.Remove("/tmp/factorio_install.tar.xz")



    // Обновляем версию в существующем экземпляре сервера
    server := factorio.GetFactorioServer()
    if err := server.RefreshVersion(); err != nil {
        log.Printf("Не удалось обновить версию: %v", err)
    }
    
    factorio.SetInstallStatus(factorio.InstallStatus{})
    wsRoom.Send(fmt.Sprintf(`{"type":"install_complete","version":"%s"}`, data.Version))
    log.Printf("[INSTALL] Complete: Factorio %s installed successfully", data.Version)
    }()

    w.WriteHeader(http.StatusAccepted)
    WriteResponse(w, map[string]string{"status": "started", "version": data.Version})
}

// GetInstallStatus returns current Factorio installation status
func GetInstallStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json;charset=UTF-8")
    status := factorio.GetInstallStatus()
    WriteResponse(w, status)
}

// CancelSyncHandler cancels the current mod sync operation
func CancelSyncHandler(w http.ResponseWriter, r *http.Request) {
    factorio.CancelSync()
    WriteResponse(w, "sync cancelled")
}

// RemoveFactorio removes the Factorio installation
func RemoveFactorio(w http.ResponseWriter, r *http.Request) {
    var resp interface{}
    defer func() { WriteResponse(w, resp) }()

    config := bootstrap.GetConfig()
    if err := os.RemoveAll(config.FactorioDir); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        resp = fmt.Sprintf("Error removing Factorio: %s", err)
        return
    }
    resp = "Factorio installation removed successfully"
    log.Println(resp)
}
