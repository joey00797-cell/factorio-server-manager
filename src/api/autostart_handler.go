package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// GET /api/autostart — получить текущее состояние
func GetAutostartHandler(w http.ResponseWriter, r *http.Request) {
	config := bootstrap.GetConfig()
	WriteResponse(w, map[string]bool{"autostart": config.AutostartServer})
}

// POST /api/autostart — сохранить состояние
func SetAutostartHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Autostart bool `json:"autostart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Читаем текущий конфиг
	config := bootstrap.GetConfig()
	confFile := config.ConfFile

	var confMap map[string]interface{}
	if raw, err := os.ReadFile(confFile); err == nil {
		json.Unmarshal(raw, &confMap)
	}
	if confMap == nil {
		confMap = make(map[string]interface{})
	}

	// Обновляем поле
	confMap["autostart_server"] = data.Autostart

	// Сохраняем
	out, err := json.MarshalIndent(confMap, "", "\t")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(confFile, out, 0644); err != nil {
		log.Printf("Error saving autostart: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Логируем - реальный автостарт произойдёт при следующем запуске контейнера
	log.Printf("Autostart will be applied on next container restart")

	log.Printf("Autostart set to: %v", data.Autostart)
	WriteResponse(w, map[string]bool{"autostart": data.Autostart})
}
