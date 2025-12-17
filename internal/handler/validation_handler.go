// internal/handler/validation_handler.go
package handler

import (
	"astralis.backend/internal/database"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func ValidateTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	var tokenID, userID int
	var isUsed bool
	var createdAt time.Time

	query := "SELECT id, user_id, is_used, created_at FROM one_time_tokens WHERE token_string = ?"
	err := database.DB.QueryRow(query, req.Token).Scan(&tokenID, &userID, &isUsed, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"status":"error", "message":"Invalid token"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"status":"error", "message":"Server error"}`, http.StatusInternalServerError)
		return
	}

	if isUsed {
		http.Error(w, `{"status":"error", "message":"Token already used"}`, http.StatusConflict)
		return
	}

	if time.Since(createdAt).Seconds() > 45 {
		http.Error(w, `{"status":"error", "message":"Token expired"}`, http.StatusRequestTimeout)
		return
	}

	// Все проверки пройдены, помечаем токен как использованный
	_, err = database.DB.Exec("UPDATE one_time_tokens SET is_used = TRUE WHERE id = ?", tokenID)
	if err != nil {
		http.Error(w, `{"status":"error", "message":"Failed to update token"}`, http.StatusInternalServerError)
		return
	}

	// Получаем имя пользователя для ответа
	var username string
	database.DB.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"username": username,
	})
}