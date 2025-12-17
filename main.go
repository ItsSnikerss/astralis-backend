package main

import (
	"astralis.backend/internal/config"
	"astralis.backend/internal/database"
	"astralis.backend/internal/handler"
	"astralis.backend/internal/middleware"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	config.LoadConfig()
	database.ConnectDB()

	r := mux.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("Получен запрос:", r.RequestURI)
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/api/register", handler.RegisterHandler).Methods("POST")
	r.HandleFunc("/api/login", handler.LoginHandler).Methods("POST")
	r.HandleFunc("/api/validate-token", handler.ValidateTokenHandler).Methods("POST")
	r.HandleFunc("/api/products", handler.GetProductsHandler).Methods("GET")
	r.HandleFunc("/api/forgot-password", handler.ForgotPasswordHandler).Methods("POST")
	r.HandleFunc("/api/reset-password", handler.ResetPasswordHandler).Methods("POST")

	protectedRoutes := r.PathPrefix("/api").Subrouter()
	protectedRoutes.Use(middleware.JWTMiddleware)

	protectedRoutes.HandleFunc("/profile", handler.ProfileHandler).Methods("GET")
	protectedRoutes.HandleFunc("/keys/activate", handler.ActivateKeyHandler).Methods("POST")

	protectedRoutes.HandleFunc("/launcher/manifest", handler.GetManifestHandler).Methods("GET")
	protectedRoutes.PathPrefix("/launcher/download/").HandlerFunc(handler.DownloadFileHandler).Methods("GET")

	adminRoutes := r.PathPrefix("/api/admin").Subrouter()
	adminRoutes.Use(middleware.JWTMiddleware, middleware.AdminMiddleware)
	adminRoutes.HandleFunc("/users", handler.AdminGetUsersHandler).Methods("GET")
	adminRoutes.HandleFunc("/keys", handler.AdminGetKeysHandler).Methods("GET")
	adminRoutes.HandleFunc("/products", handler.AdminCreateProductHandler).Methods("POST")
	adminRoutes.HandleFunc("/keys", handler.AdminCreateKeyHandler).Methods("POST")
	adminRoutes.HandleFunc("/users/{id}/status", handler.AdminUpdateUserStatusHandler).Methods("PATCH")
	adminRoutes.HandleFunc("/users/{id}", handler.AdminDeleteUserHandler).Methods("DELETE")
	adminRoutes.HandleFunc("/products/{id}", handler.AdminUpdateProductHandler).Methods("PUT")
	adminRoutes.HandleFunc("/products/{id}", handler.AdminDeleteProductHandler).Methods("DELETE")

	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:3000", "null"}) 
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})
	corsRouter := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r)

	server := &http.Server{Addr: config.Cfg.Port, Handler: corsRouter}

	go func() {
		log.Println("Starting backend server on http://localhost" + config.Cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", config.Cfg.Port, err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}
	log.Println("Server exited properly")
}