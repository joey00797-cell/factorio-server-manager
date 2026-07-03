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
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}

	savesDir := serverSavesDir(server)
	savesList, err := factorio.ListSavesInDir(savesDir)
	if err != nil {
		resp = fmt.Sprintf("Error listing save files: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if withLatest {
		savesList, err = factorio.ListSavesWithLatestInDir(savesDir)
		if err != nil {
			resp = fmt.Sprintf("Error listing save files: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	resp = savesList
}

func DLSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	server, ok := serverFromRequest(r)
	if !ok {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}
	vars := mux.Vars(r)
	save := vars["save"]
	saveName, err := factorio.SavePathInDir(serverSavesDir(server), save)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid save name: %s", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(saveName)))
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
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, saveFile := range r.MultipartForm.File["savefile"] {
		ext := filepath.Ext(saveFile.Filename)
		if !strings.EqualFold(ext, ".zip") {
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

		savesDir := serverSavesDir(server)
		savePath, err := factorio.SavePathInDir(savesDir, saveFile.Filename)
		if err != nil {
			resp = fmt.Sprintf("Invalid save filename {%s}: %s", saveFile.Filename, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(savesDir, 0755); err != nil {
			resp = fmt.Sprintf("Error creating saves directory: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		out, err := os.Create(savePath)
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

	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}

	savesDir := serverSavesDir(server)
	save, err := factorio.FindSaveInDir(savesDir, name)
	if err != nil {
		resp = fmt.Sprintf("Error finding save {%s}: %s", name, err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = save.RemoveFromDir(savesDir)
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
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if manager := factorio.GetServerManager(); manager != nil {
		if err := manager.EnsureServerVersion(server); err != nil {
			resp = fmt.Sprintf("Error preparing Factorio install for server %s: %s", server.ID, err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	saveFile, saveName, err := factorio.SavePathForCreateInDir(serverSavesDir(server), saveName)
	if err != nil {
		resp = fmt.Sprintf("Invalid save filename {%s}: %s", saveName, err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cmdOut, err := factorio.CreateSaveWithBinary(saveFile, serverBinary(server))
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

// LogTail returns last lines of this server's Factorio log.
func LogTail(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
	for _, logFile := range serverLogFiles(server) {
		if info, statErr := os.Stat(logFile); statErr != nil || info.IsDir() {
			continue
		}
		resp, err = factorio.TailLogFile(logFile)
		if err != nil {
			resp = fmt.Sprintf("Could not tail %s: %s", logFile, err)
			return
		}
		return
	}
	resp = []string{}
}

// FSMLogTail returns Factorio Server Manager process logs, not per-server logs.
func FSMLogTail(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	logFile := bootstrap.FSMLogFile()
	if _, statErr := os.Stat(logFile); os.IsNotExist(statErr) {
		resp = []string{}
		return
	}
	resp, err = factorio.TailLogFile(logFile)
	if err != nil {
		resp = fmt.Sprintf("Could not tail %s: %s", logFile, err)
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
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = factorio.EnsureInstanceConfig(server.Paths)
	configContents, err := factorio.LoadConfig(serverConfigFile(server))
	if err != nil {
		log.Printf("config.ini not available: %s", err)
		resp = map[string]interface{}{}
		return
	}

	resp = configContents

	log.Printf("Sent config.ini response")
}

func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var data map[string]map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		resp = fmt.Sprintf("Error parsing config.ini JSON: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := factorio.SaveConfig(serverConfigFile(server), data); err != nil {
		resp = fmt.Sprintf("Error saving config.ini: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if server.Running {
		server.PendingRestart = true
		if manager := factorio.GetServerManager(); manager != nil {
			_ = manager.UpdateServer(server)
		}
	}
	resp = "config.ini saved"
}

func StartServer(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}
	server, ok := serverFromRequest(r)
	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	var startData struct {
		BindIP   string `json:"bindip"`
		Savefile string `json:"savefile"`
		Port     int    `json:"port"`
	}
	err = json.Unmarshal(body, &startData)
	if err != nil {
		resp = fmt.Sprintf("Error unmarshalling server settings JSON: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if startData.BindIP != "" {
		server.BindIP = startData.BindIP
	}
	if startData.Port != 0 {
		server.Port = startData.Port
	}
	server.Savefile = startData.Savefile

	// Check if savefile was submitted with request to start server.
	if server.Savefile == "" {
		resp = "Error starting Factorio server: No save file provided"
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(server.Savefile, "Load Latest") {
		save, err := factorio.FindSaveInDir(serverSavesDir(server), server.Savefile)
		if err != nil {
			resp = fmt.Sprintf("Error starting Factorio server: invalid save file {%s}: %s", server.Savefile, err)
			log.Println(resp)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		server.Savefile = save.Name
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
	if manager := factorio.GetServerManager(); manager != nil {
		_ = manager.UpdateServer(server)
	}
	log.Println(resp)
}

func StopServer(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
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
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
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
	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		WriteResponse(w, "server not found")
		return
	}
	defer func() {
		WriteResponse(w, server)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
}

func FactorioVersion(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	server, ok := serverFromRequest(r)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		resp["error"] = "server not found"
		return
	}
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
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
	data, err := os.ReadFile(serverSettingsFile(server))
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
		server, _ := serverFromRequest(r)
		existingRaw, _ := os.ReadFile(serverSettingsFile(server))
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
	// Читаем текущий файл
	server, ok := serverFromRequest(r)
	if !ok {
		resp = "server not found"
		w.WriteHeader(http.StatusNotFound)
		return
	}
	existingData, err2 := os.ReadFile(serverSettingsFile(server))
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
	server.Settings = currentSettings

	settings, err := json.MarshalIndent(currentSettings, "", "  ")
	if err != nil {
		resp = fmt.Sprintf("Failed to marshal server settings: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = ioutil.WriteFile(serverSettingsFile(server), settings, 0644)
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

		err = ioutil.WriteFile(serverAdminFile(server), admins, 0664)
		if err != nil {
			resp = fmt.Sprintf("Failed to save admins: %s", err)
			log.Println(resp)
			return
		}
	}

	resp = fmt.Sprintf("Settings successfully saved")
	if server.Running {
		server.PendingRestart = true
	}
	if manager := factorio.GetServerManager(); manager != nil {
		_ = manager.UpdateServer(server)
	}
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

func DownloadedVersionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server manager not initialized"}`))
		return
	}
	versions := manager.ListDownloadedVersions()
	json.NewEncoder(w).Encode(map[string]interface{}{"versions": versions})
}

func DeleteDownloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server manager not initialized"}`))
		return
	}
	vars := mux.Vars(r)
	version := vars["version"]
	if err := manager.DeleteDownload(version); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
		return
	}
	w.Write([]byte(`{"ok":true}`))
}

func InstalledVersionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server manager not initialized"}`))
		return
	}
	versions := manager.ListInstalledVersions()
	json.NewEncoder(w).Encode(map[string]interface{}{"versions": versions})
}

func InstallFactorio(w http.ResponseWriter, r *http.Request) {
	var resp interface{}
	defer func() { WriteResponse(w, resp) }()

	var data struct {
		Version string `json:"version"`
	}
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &data)

	if data.Version == "" {
		data.Version = "stable"
	}

	manager := factorio.GetServerManager()
	if manager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = "server manager is not initialized"
		return
	}
	installedVersion, err := manager.EnsureVersionInstalled(data.Version)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error installing Factorio: %s", err)
		return
	}
	resp = fmt.Sprintf("Factorio %s installed successfully", installedVersion)
	log.Println(resp)
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
