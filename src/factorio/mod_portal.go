package factorio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ModPortalStruct struct {
	DownloadsCount int    `json:"downloads_count"`
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	Releases       []struct {
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		InfoJSON    struct {
			FactorioVersion Version `json:"factorio_version"`
		} `json:"info_json"`
		ReleasedAt    time.Time `json:"released_at"`
		Sha1          string    `json:"sha1"`
		Version       Version   `json:"version"`
		Compatibility bool      `json:"compatibility"`
	} `json:"releases"`
	Summary string `json:"summary"`
	Title   string `json:"title"`
}

// get all mods uploaded to the factorio modPortal
func ModPortalList() (interface{}, error, int) {
	req, err := http.NewRequest(http.MethodGet, "https://mods.factorio.com/api/mods?page_size=max", nil)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(text)), resp.StatusCode
	}

	var jsonVal interface{}
	err = json.Unmarshal(text, &jsonVal)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	return jsonVal, nil, resp.StatusCode
}

// get the details (mod-info, releases, etc.) from a specific mod from the modPortal
func ModPortalModDetails(modId string) (ModPortalStruct, error, int) {
	var mod ModPortalStruct

	req, err := http.NewRequest(http.MethodGet, "https://mods.factorio.com/api/mods/"+modId, nil)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	err = json.Unmarshal(text, &mod)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return ModPortalStruct{}, errors.New(string(text)), resp.StatusCode
	}

	server := GetFactorioServer()

	installedBaseVersion := Version{}
	_ = installedBaseVersion.UnmarshalText([]byte(server.BaseModVersion))
	requiredVersion := NilVersion

	for key, release := range mod.Releases {
		requiredVersion = release.InfoJSON.FactorioVersion
		release.Compatibility = installedBaseVersion.Compatible(requiredVersion, ">=")
		mod.Releases[key] = release
	}

	return mod, nil, resp.StatusCode
}

// Log the user into factorio, so mods can be downloaded
func FactorioLogin(username string, password string) (error, int) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		return errors.New("username and password are required"), http.StatusBadRequest
	}

	resp, err := http.PostForm("https://auth.factorio.com/api-login",
		url.Values{
			"api_version":            {"6"},
			"require_game_ownership": {"true"},
			"username":               {username},
			"password":               {password},
		})

	if err != nil {
		return err, http.StatusInternalServerError
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bodyBytes)), resp.StatusCode
	}

	var authResponse struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Error    string `json:"error"`
		Message  string `json:"message"`
	}
	if err = json.Unmarshal(bodyBytes, &authResponse); err != nil {
		return err, http.StatusInternalServerError
	}

	if authResponse.Error != "" {
		if authResponse.Message != "" {
			return errors.New(authResponse.Message), http.StatusUnauthorized
		}
		return errors.New(authResponse.Error), http.StatusUnauthorized
	}

	if authResponse.Token == "" || authResponse.Username == "" {
		return errors.New("Factorio login did not return a token"), http.StatusBadGateway
	}

	credentials := Credentials{
		Username: authResponse.Username,
		Userkey:  authResponse.Token,
	}

	if err := credentials.Validate(); err != nil {
		if err == ErrInvalidFactorioCredentials {
			return errors.New("Factorio accepted the login, but the mod portal rejected the returned token"), http.StatusForbidden
		}
		return err, http.StatusBadGateway
	}

	err = credentials.Save()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func FactorioLoginWithPasswordOrToken(username string, passwordOrToken string) (error, int) {
	username = strings.TrimSpace(username)
	passwordOrToken = strings.TrimSpace(passwordOrToken)

	if username == "" || passwordOrToken == "" {
		return errors.New("username and password/token are required"), http.StatusBadRequest
	}

	tokenErr, tokenStatus := FactorioLoginWithToken(username, passwordOrToken)
	if tokenErr == nil {
		return nil, tokenStatus
	}
	if tokenStatus != http.StatusForbidden {
		return tokenErr, tokenStatus
	}

	passwordErr, passwordStatus := FactorioLogin(username, passwordOrToken)
	if passwordErr == nil {
		return nil, passwordStatus
	}

	return fmt.Errorf("Factorio rejected this as both a token and a password: %s", passwordErr), passwordStatus
}

// FactorioLoginWithToken validates and saves credentials using a token from factorio.com/profile.
func FactorioLoginWithToken(username string, token string) (error, int) {
	username = strings.TrimSpace(username)
	token = strings.TrimSpace(token)

	if username == "" || token == "" {
		return errors.New("username and token are required"), http.StatusBadRequest
	}

	credentials := Credentials{
		Username: username,
		Userkey:  token,
	}

	if err := credentials.Validate(); err != nil {
		if err == ErrInvalidFactorioCredentials {
			return fmt.Errorf("Factorio rejected this username/token pair"), http.StatusForbidden
		}
		return err, http.StatusBadGateway
	}

	err := credentials.Save()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func FactorioLoginStatus() (bool, error, int) {
	var credentials Credentials
	status, err := credentials.Load()
	if err != nil {
		return false, err, http.StatusInternalServerError
	}
	if !status {
		return false, nil, http.StatusOK
	}

	err = credentials.Validate()
	if err == nil {
		return true, nil, http.StatusOK
	}
	if err == ErrInvalidFactorioCredentials {
		_ = credentials.Del()
		return false, nil, http.StatusOK
	}

	return false, err, http.StatusBadGateway
}
