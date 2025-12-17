package handler

import (
	"astralis.backend/internal/database"
	"astralis.backend/internal/model"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
)

func AdminGetUsersHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := query.Get("search")
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	searchPattern := "%" + search + "%"

	var totalUsers int
	countQuery := "SELECT COUNT(*) FROM users WHERE username LIKE ? OR email LIKE ?"
	err = database.DB.QueryRow(countQuery, searchPattern, searchPattern).Scan(&totalUsers)
	if err != nil {
		http.Error(w, `{"error":"Failed to count users"}`, http.StatusInternalServerError)
		return
	}
	totalPages := int(math.Ceil(float64(totalUsers) / float64(limit)))
	if totalPages == 0 && totalUsers > 0 {
		totalPages = 1
	}

	selectQuery := "SELECT id, username, email, role, subscription_expires_at, is_banned, hwid FROM users WHERE username LIKE ? OR email LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?"
	rows, err := database.DB.Query(selectQuery, searchPattern, searchPattern, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"Failed to query users"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := make([]model.UserForAdmin, 0)
	for rows.Next() {
		var user model.UserForAdmin
		var email, role, hwid sql.NullString
		var expiresAt sql.NullTime
		if err := rows.Scan(&user.ID, &user.Username, &email, &role, &expiresAt, &user.IsBanned, &hwid); err != nil {
			http.Error(w, `{"error":"Failed to scan user row"}`, http.StatusInternalServerError)
			return
		}
		if email.Valid { user.Email = email.String }
		if role.Valid { user.Role = role.String } else { user.Role = "user" }
		if expiresAt.Valid { user.SubscriptionExpiresAt = &expiresAt.Time }
		if hwid.Valid { user.Hwid = &hwid.String }
		users = append(users, user)
	}

	response := model.PaginatedUsersResponse{
		Users:       users,
		TotalPages:  totalPages,
		CurrentPage: page,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AdminGetKeysHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	var totalKeys int
	err = database.DB.QueryRow("SELECT COUNT(*) FROM activation_keys").Scan(&totalKeys)
	if err != nil {
		http.Error(w, `{"error":"Failed to count keys"}`, http.StatusInternalServerError)
		return
	}
	totalPages := int(math.Ceil(float64(totalKeys) / float64(limit)))
	if totalPages == 0 && totalKeys > 0 {
		totalPages = 1
	}

	rows, err := database.DB.Query("SELECT id, key_string, duration_days, is_used, used_by_user_id, used_at FROM activation_keys ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		http.Error(w, `{"error":"Failed to query keys"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	keys := make([]model.KeyForAdmin, 0)
	for rows.Next() {
		var key model.KeyForAdmin
		var usedBy sql.NullInt64
		var usedAt sql.NullTime

		if err := rows.Scan(&key.ID, &key.KeyString, &key.DurationDays, &key.IsUsed, &usedBy, &usedAt); err != nil {
			http.Error(w, `{"error":"Failed to scan key row"}`, http.StatusInternalServerError)
			return
		}
		if usedBy.Valid {
			tempID := int(usedBy.Int64)
			key.UsedByUserID = &tempID
		}
		if usedAt.Valid {
			key.UsedAt = &usedAt.Time
		}
		keys = append(keys, key)
	}
	
	response := model.PaginatedKeysResponse{
		Keys:        keys,
		TotalPages:  totalPages,
		CurrentPage: page,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AdminCreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	query := "INSERT INTO products (name, description, price, is_featured) VALUES (?, ?, ?, ?)"
	_, err := database.DB.Exec(query, p.Name, p.Description, p.Price, p.IsFeatured)
	if err != nil {
		http.Error(w, `{"error":"Failed to create product"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Product created successfully"})
}