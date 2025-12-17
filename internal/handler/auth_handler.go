package handler

import (
	"astralis.backend/internal/database"
	"astralis.backend/internal/middleware"
	"astralis.backend/internal/model"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"
)

var jwtKey = []byte("your_very_secret_key_that_is_long_and_secure")
var turnstileSecretKey = "0x4AAAAAAAB_en5I_dmjXCe8cEAPUJL1DVY"

type TurnstileResponse struct {
	Success bool `json:"success"`
}

func generateOneTimeToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		model.User
		TurnstileToken string `json:"turnstileToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.TurnstileToken == "" {
		http.Error(w, `{"error":"CAPTCHA verification failed"}`, http.StatusBadRequest)
		return
	}

	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {turnstileSecretKey},
		"response": {req.TurnstileToken},
	})
	if err != nil {
		http.Error(w, `{"error":"Failed to verify CAPTCHA"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var turnstileResp TurnstileResponse
	json.Unmarshal(body, &turnstileResp)

	if !turnstileResp.Success {
		http.Error(w, `{"error":"CAPTCHA verification failed, please try again"}`, http.StatusForbidden)
		return
	}
	
	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"All fields are required"}`, http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.Password) < 8 {
		http.Error(w, `{"error":"Password must be at least 8 characters long"}`, http.StatusBadRequest)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	
	query := "INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, 'user')"
	if _, err := database.DB.Exec(query, req.Username, req.Email, hashedPassword); err != nil {
		http.Error(w, `{"error":"Username or email may already exist."}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Hwid     string `json:"hwid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}
	var storedHash string
	var userRole, storedHwid sql.NullString
	var userID int
	var isBanned bool
	query := "SELECT id, password_hash, role, is_banned, hwid FROM users WHERE username = ?"
	err := database.DB.QueryRow(query, creds.Username).Scan(&userID, &storedHash, &userRole, &isBanned, &storedHwid)
	if err != nil {
		http.Error(w, `{"error":"Invalid username or password"}`, http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(creds.Password)); err != nil {
		http.Error(w, `{"error":"Invalid username or password"}`, http.StatusUnauthorized)
		return
	}
	if isBanned {
		http.Error(w, `{"error":"This account is banned"}`, http.StatusForbidden)
		return
	}

	if creds.Hwid != "" {
		if storedHwid.Valid && storedHwid.String != creds.Hwid {
			http.Error(w, `{"error":"HWID mismatch"}`, http.StatusConflict)
			return
		} else if !storedHwid.Valid {
			database.DB.Exec("UPDATE users SET hwid = ? WHERE id = ?", creds.Hwid, userID)
		}
	}
	

	oneTimeToken, err := generateOneTimeToken()
	if err != nil {
		http.Error(w, `{"error":"Could not generate session token"}`, http.StatusInternalServerError)
		return
	}
	_, err = database.DB.Exec("INSERT INTO one_time_tokens (token_string, user_id) VALUES (?, ?)", oneTimeToken, userID)
	if err != nil {
		http.Error(w, `{"error":"Could not save session token"}`, http.StatusInternalServerError)
		return
	}

	finalRole := "user"
	if userRole.Valid {
		finalRole = userRole.String
	}
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.AppClaims{
		Role: finalRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Subject:   strconv.Itoa(userID),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtTokenString, err := jwtToken.SignedString(jwtKey)
	if err != nil {
		http.Error(w, `{"error":"Could not create auth token"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"session_token": oneTimeToken,
		"token":         jwtTokenString,
	})
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r)
	if !ok {
		http.Error(w, `{"error":"Could not retrieve user claims"}`, http.StatusInternalServerError)
		return
	}

	userID := claims.Subject

	var username, email string
	var role, hwid sql.NullString
	var subscriptionExpiresAt sql.NullTime

	query := "SELECT username, email, role, subscription_expires_at, hwid FROM users WHERE id = ?"
	err := database.DB.QueryRow(query, userID).Scan(&username, &email, &role, &subscriptionExpiresAt, &hwid)
	if err != nil {
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}

	finalRole := "user"
	if role.Valid {
		finalRole = role.String
	}

	response := map[string]interface{}{
		"id":       userID,
		"username": username,
		"email":    email,
		"role":     finalRole,
	}

	if subscriptionExpiresAt.Valid {
		response["subscription_expires_at"] = subscriptionExpiresAt.Time
	} else {
		response["subscription_expires_at"] = nil
	}

	if hwid.Valid {
		response["hwid"] = hwid.String
	} else {
		response["hwid"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}