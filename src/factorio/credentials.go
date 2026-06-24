package factorio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

var ErrInvalidFactorioCredentials = errors.New("invalid Factorio username or token")

type Credentials struct {
	Username string `json:"username"`
	Userkey  string `json:"userkey"`
}

func (credentials *Credentials) Save() error {
	var err error
	config := bootstrap.GetConfig()

	credentials.Username = strings.TrimSpace(credentials.Username)
	credentials.Userkey = strings.TrimSpace(credentials.Userkey)

	credentialsJson, err := json.Marshal(credentials)
	if err != nil {
		log.Printf("error mashalling the credentials: %s", err)
		return err
	}

	err = ioutil.WriteFile(config.FactorioCredentialsFile, credentialsJson, 0664)
	if err != nil {
		log.Printf("error on saving the credentials. %s", err)
		return err
	}

	return nil
}

func (credentials *Credentials) Load() (bool, error) {
	var err error
	config := bootstrap.GetConfig()
	if _, err := os.Stat(config.FactorioCredentialsFile); os.IsNotExist(err) {
		return false, nil
	}

	fileBytes, err := ioutil.ReadFile(config.FactorioCredentialsFile)
	if err != nil {
		credentials.Del()
		log.Printf("error reading CredentialsFile: %s", err)
		return false, err
	}

	err = json.Unmarshal(fileBytes, credentials)
	if err != nil {
		credentials.Del()
		log.Printf("error on unmarshal credentials_file: %s", err)
		return false, err
	}

	credentials.Username = strings.TrimSpace(credentials.Username)
	credentials.Userkey = strings.TrimSpace(credentials.Userkey)

	if credentials.Userkey != "" && credentials.Username != "" {
		return true, nil
	} else {
		credentials.Del()
		return false, errors.New("incredients incomplete")
	}
}

func (credentials *Credentials) Del() error {
	var err error
	config := bootstrap.GetConfig()
	err = os.Remove(config.FactorioCredentialsFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		log.Printf("error delete the credentialfile: %s", err)
		return err
	}

	return nil
}

func (credentials *Credentials) DownloadURL(downloadPath string) string {
	downloadURL, err := url.Parse("https://mods.factorio.com" + downloadPath)
	if err != nil {
		return "https://mods.factorio.com" + downloadPath
	}

	query := downloadURL.Query()
	query.Set("username", strings.TrimSpace(credentials.Username))
	query.Set("token", strings.TrimSpace(credentials.Userkey))
	downloadURL.RawQuery = query.Encode()

	return downloadURL.String()
}

func (credentials *Credentials) Validate() error {
	credentials.Username = strings.TrimSpace(credentials.Username)
	credentials.Userkey = strings.TrimSpace(credentials.Userkey)

	if credentials.Username == "" || credentials.Userkey == "" {
		return ErrInvalidFactorioCredentials
	}

	validateURL, err := url.Parse("https://mods.factorio.com/api/bookmarks")
	if err != nil {
		return err
	}

	query := validateURL.Query()
	query.Set("username", credentials.Username)
	query.Set("token", credentials.Userkey)
	validateURL.RawQuery = query.Encode()

	resp, err := http.Get(validateURL.String())
	if err != nil {
		return fmt.Errorf("validating Factorio credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusForbidden {
		return ErrInvalidFactorioCredentials
	}

	return fmt.Errorf("validating Factorio credentials returned status %d", resp.StatusCode)
}
