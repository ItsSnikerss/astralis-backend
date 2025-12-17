package handler

import (
	"astralis.backend/internal/database"
	"astralis.backend/internal/model"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func AdminUpdateUserStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, `{"error":"Invalid user ID"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		IsBanned  *bool `json:"is_banned"`
		ResetSub  *bool `json:"reset_subscription"`
		ResetHwid *bool `json:"reset_hwid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	
	if req.IsBanned != nil {
		_, err = database.DB.Exec("UPDATE users SET is_banned = ? WHERE id = ?", *req.IsBanned, userID)
	}
	if req.ResetSub != nil && *req.ResetSub {
		_, err = database.DB.Exec("UPDATE users SET subscription_expires_at = NULL WHERE id = ?", userID)
	}
    if req.ResetHwid != nil && *req.ResetHwid {
        _, err = database.DB.Exec("UPDATE users SET hwid = NULL WHERE id = ?", userID)
    }

	if err != nil {
		http.Error(w, `{"error":"Failed to update user"}`, http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
}

func AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, `{"error":"Invalid user ID"}`, http.StatusBadRequest)
		return
	}

	_, err = database.DB.Exec("DELETE FROM activation_keys WHERE used_by_user_id = ?", userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to delete user's keys"}`, http.StatusInternalServerError)
		return
	}
	
	_, err = database.DB.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to delete user"}`, http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

func generateActivationKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X-%X-%X-%X", bytes[0:4], bytes[4:8], bytes[8:12], bytes[12:16]), nil
}

func AdminCreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DurationDays int `json:"duration_days"`
		Quantity     int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.DurationDays < 0 {
		http.Error(w, `{"error":"Duration cannot be negative"}`, http.StatusBadRequest)
		return
	}
	if req.Quantity <= 0 || req.Quantity > 100 {
		http.Error(w, `{"error":"Quantity must be between 1 and 100"}`, http.StatusBadRequest)
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, `{"error":"Failed to start transaction"}`, http.StatusInternalServerError)
		return
	}

	query := "INSERT INTO activation_keys (key_string, duration_days) VALUES (?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		http.Error(w, `{"error":"Failed to prepare statement"}`, http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	for i := 0; i < req.Quantity; i++ {
		newKey, err := generateActivationKey()
		if err != nil {
			tx.Rollback()
			http.Error(w, `{"error":"Failed to generate key"}`, http.StatusInternalServerError)
			return
		}
		_, err = stmt.Exec(newKey, req.DurationDays)
		if err != nil {
			tx.Rollback()
			http.Error(w, `{"error":"Failed to save key to database"}`, http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to commit transaction"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("%d keys created successfully", req.Quantity)})
}


func AdminUpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, `{"error":"Invalid product ID"}`, http.StatusBadRequest)
		return
	}

	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	query := "UPDATE products SET name = ?, description = ?, price = ?, is_featured = ? WHERE id = ?"
	_, err = database.DB.Exec(query, p.Name, p.Description, p.Price, p.IsFeatured, productID)
	if err != nil {
		http.Error(w, `{"error":"Failed to update product"}`, http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Product updated successfully"})
}

func AdminDeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, `{"error":"Invalid product ID"}`, http.StatusBadRequest)
		return
	}

	_, err = database.DB.Exec("DELETE FROM products WHERE id = ?", productID)
	if err != nil {
		http.Error(w, `{"error":"Failed to delete product"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Product deleted successfully"})
}