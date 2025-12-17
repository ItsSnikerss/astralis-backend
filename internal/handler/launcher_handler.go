package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

type GameFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

func GetManifestHandler(w http.ResponseWriter, r *http.Request) {
	filePath := "./game_files/client.zip"
	
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("ОШИБКА: Не могу открыть файл для манифеста: %v", err)
		http.Error(w, `{"error":"Файл на сервере не найден"}`, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Printf("ОШИБКА: Не могу посчитать хеш файла: %v", err)
		http.Error(w, `{"error":"Ошибка чтения файла на сервере"}`, http.StatusInternalServerError)
		return
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("ОШИБКА: Не могу получить информацию о файле: %v", err)
		http.Error(w, `{"error":"Ошибка чтения файла на сервере"}`, http.StatusInternalServerError)
		return
	}
	size := fileInfo.Size()

	manifest := []GameFile{
		{
			Path: "client.zip",
			Hash: hash,
			Size: size,
			URL:  "/api/launcher/download/client.zip",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	fs := http.StripPrefix("/api/launcher/download/", http.FileServer(http.Dir("./game_files/")))
	fs.ServeHTTP(w, r)
}