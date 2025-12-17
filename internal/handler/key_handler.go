package handler

import (
	"astralis.backend/internal/database"
	"astralis.backend/internal/middleware"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func ActivateKeyHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r)
	if !ok {
		http.Error(w, `{"error":"Could not retrieve user claims"}`, http.StatusInternalServerError)
		return
	}
	userID := claims.Subject

	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, `{"error":"Key is required"}`, http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"Server error"}`, http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var keyID, durationDays int
	var isUsed bool
	query := "SELECT id, duration_days, is_used FROM activation_keys WHERE key_string = ? FOR UPDATE"
	err = tx.QueryRow(query, req.Key).Scan(&keyID, &durationDays, &isUsed)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Invalid activation key"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"Server error on key lookup"}`, http.StatusInternalServerError)
		return
	}

	if isUsed {
		http.Error(w, `{"error":"Activation key has already been used"}`, http.StatusConflict)
		return
	}
	
	var message string
	if durationDays > 0 {
		var currentSub sql.NullTime
		err := tx.QueryRow("SELECT subscription_expires_at FROM users WHERE id = ?", userID).Scan(&currentSub)
		if err != nil {
			http.Error(w, `{"error":"Failed to get current subscription"}`, http.StatusInternalServerError)
			return
		}
		
		newExpiryDate := time.Now()
		if currentSub.Valid && currentSub.Time.After(newExpiryDate) {
			newExpiryDate = currentSub.Time
		}
		newExpiryDate = newExpiryDate.Add(time.Duration(durationDays) * 24 * time.Hour)
		
		_, err = tx.Exec("UPDATE users SET subscription_expires_at = ? WHERE id = ?", newExpiryDate, userID)
		if err != nil {
			http.Error(w, `{"error":"Failed to update user subscription"}`, http.StatusInternalServerError)
			return
		}
		message = fmt.Sprintf("Subscription extended for %d days", durationDays)

	} else {
		_, err := tx.Exec("UPDATE users SET hwid = NULL WHERE id = ?", userID)
		if err != nil {
			http.Error(w, `{"error":"Failed to reset HWID"}`, http.StatusInternalServerError)
			return
		}
		message = "HWID has been successfully reset"
	}

	_, err = tx.Exec("UPDATE activation_keys SET is_used = TRUE, used_by_user_id = ?, used_at = ? WHERE id = ?", userID, time.Now(), keyID)
	if err != nil {
		http.Error(w, `{"error":"Failed to update key status"}`, http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to commit transaction"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}