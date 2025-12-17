// internal/handler/password_reset_handler.go
package handler

import (
	"astralis.backend/internal/config"
	"astralis.backend/internal/database"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
	"unicode/utf8"
)

func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	var userID int
	var username string
	err := database.DB.QueryRow("SELECT id, username FROM users WHERE email = ?", req.Email).Scan(&userID, &username)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "If an account with that email exists, a password reset link has been sent."})
		return
	}

	token, err := generateResetToken()
	if err != nil {
		http.Error(w, `{"error":"Could not generate token"}`, http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	_, err = database.DB.Exec(
		"INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID, token, expiresAt,
	)
	if err != nil {
		http.Error(w, `{"error":"Could not save token"}`, http.StatusInternalServerError)
		return
	}

	resetLink := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)

	emailHTML := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="ru">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Восстановление доступа</title>
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #06080f; color: #e0e0e0; margin: 0; padding: 20px; }
				.container { max-width: 600px; margin: 0 auto; background-color: #101422; border: 1px solid #366a98; border-radius: 12px; padding: 40px; text-align: center; }
				.logo { font-size: 24px; font-weight: bold; color: #fff; margin-bottom: 30px; }
				.logo span { color: #59a6e4; }
				h1 { color: #ffffff; font-size: 28px; margin-bottom: 20px; }
				p { font-size: 16px; line-height: 1.6; color: #a0a0a0; margin-bottom: 30px; }
				.button { display: inline-block; background-color: #59a6e4; color: #ffffff !important; padding: 15px 30px; border-radius: 8px; text-decoration: none; font-size: 16px; font-weight: bold; }
				.footer-text { font-size: 12px; color: #606060; margin-top: 30px; }
				.footer-text a { color: #59a6e4; text-decoration: none; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="logo">Astralis<span>Client</span></div>
				<h1>Восстановление доступа</h1>
				<p>Чтобы восстановить доступ к вашему аккаунту <strong>%s</strong>, нажмите на кнопку ниже.</p>
				<a href="%s" class="button">Восстановить доступ</a>
				<p class="footer-text">Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.<br>Не можете нажать на кнопку? Скопируйте и вставьте эту ссылку в браузер:<br><a href="%s">%s</a></p>
			</div>
		</body>
		</html>
	`, username, resetLink, resetLink, resetLink)

	payload := map[string]string{
		"from":    config.Cfg.EmailSender,
		"to":      req.Email,
		"subject": "Восстановление доступа к аккаунту Astralis",
		"html":    emailHTML,
	}
	jsonPayload, _ := json.Marshal(payload)

	emailReq, _ := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonPayload))
	emailReq.Header.Set("Authorization", "Bearer "+config.Cfg.ResendApiKey)
	emailReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(emailReq)
	if err != nil || resp.StatusCode >= 400 {
		// Production: log this error. For now, we do nothing.
	} else {
		defer resp.Body.Close()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "If an account with that email exists, a password reset link has been sent."})
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(req.NewPassword) < 8 {
		http.Error(w, `{"error":"Password must be at least 8 characters long"}`, http.StatusBadRequest)
		return
	}

	var userID int
	var expiresAt time.Time
	var isUsed bool
	query := "SELECT user_id, expires_at, is_used FROM password_reset_tokens WHERE token = ?"
	err := database.DB.QueryRow(query, req.Token).Scan(&userID, &expiresAt, &isUsed)
	if err != nil {
		http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusBadRequest)
		return
	}

	if isUsed || time.Now().After(expiresAt) {
		http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"Failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	tx, _ := database.DB.Begin()
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hashedPassword, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to update password"}`, http.StatusInternalServerError)
		return
	}
	_, err = tx.Exec("UPDATE password_reset_tokens SET is_used = TRUE WHERE token = ?", req.Token)
	if err != nil {
		http.Error(w, `{"error":"Failed to invalidate token"}`, http.StatusInternalServerError)
		return
	}
	
	if err := tx.Commit(); err != nil {
		http.Error(w, `{"error":"Failed to commit transaction"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password has been reset successfully."})
}