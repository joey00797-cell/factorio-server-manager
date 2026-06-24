package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/gorilla/mux"
)

// Встроенные языки (всегда доступны)
var builtinLocales = map[string]string{
	"en": "English",
	"ru": "Русский",
	"zh": "中文",
}

func getLocalesDir() string {
	dir := bootstrap.GetConfig().LocalesDir
	if dir == "" {
		dir = "/opt/fsm-data/locales"
	}
	return dir
}

// GET /api/locales/list — список всех доступных языков
func LocalesListHandler(w http.ResponseWriter, r *http.Request) {
	type LangEntry struct {
		Code    string `json:"code"`
		Name    string `json:"name"`
		Builtin bool   `json:"builtin"`
	}

	result := []LangEntry{}
	for code, name := range builtinLocales {
		result = append(result, LangEntry{Code: code, Name: name, Builtin: true})
	}

	// Добавляем кастомные из /opt/fsm-data/locales/
	dir := getLocalesDir()
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			code := strings.TrimSuffix(e.Name(), ".json")
			if _, builtin := builtinLocales[code]; !builtin {
				// Читаем _lang_name из файла если есть
				name := code
				if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
					var js map[string]interface{}
					if json.Unmarshal(data, &js) == nil {
						if n, ok := js["_lang_name"].(string); ok && n != "" {
							name = n
						}
					}
				}
				result = append(result, LangEntry{Code: code, Name: name, Builtin: false})
			}
		}
	}

	WriteResponse(w, result)
}

// GET /api/locales/template — скачать en.json как шаблон
func LocalesTemplateHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем встроенный en.json из app/locales/
	data, err := os.ReadFile("./app/locales/en.json")
	if err != nil {
		log.Printf("Error reading en.json: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"en.json\"")
	w.Write(data)
}

// GET /api/locales/{lang} — отдать локаль (встроенную или кастомную)
func LocalesGetHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lang := vars["lang"]

	// Сначала ищем кастомную
	customPath := filepath.Join(getLocalesDir(), lang+".json")
	if data, err := os.ReadFile(customPath); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		return
	}

	// Потом встроенную
	builtinPath := filepath.Join("./app/locales", lang+".json")
	data, err := os.ReadFile(builtinPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// POST /api/locales/upload — загрузить новую локаль
func LocalesUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(5 << 20) // 5MB

	file, header, err := r.FormFile("locale")
	if err != nil {
		log.Printf("Error reading locale file: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		WriteResponse(w, "No file provided")
		return
	}
	defer file.Close()

	// Проверяем что это .json
	if !strings.HasSuffix(header.Filename, ".json") {
		w.WriteHeader(http.StatusBadRequest)
		WriteResponse(w, "Only .json files allowed")
		return
	}

	// Валидируем JSON
	data, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var js map[string]interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		WriteResponse(w, "Invalid JSON")
		return
	}

	// Сохраняем
	dir := getLocalesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(dir, header.Filename)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		log.Printf("Error saving locale: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("Locale uploaded: %s", header.Filename)
	WriteResponse(w, "OK")
}
