package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rtcs/internal/cache"
	"rtcs/internal/config"
	"rtcs/internal/middleware"
	"rtcs/internal/repository"
	"rtcs/internal/service"
	"rtcs/internal/transport"
	httptransport "rtcs/internal/transport/http"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile)
}

func main() {
	log.Printf("Starting server...")

	// Initialize configuration
	cfg := config.Get()
	log.Printf("Configuration loaded")

	// Load OAuth configuration
	oauthCfg, err := config.LoadOAuthConfig()
	if err != nil {
		log.Fatalf("Failed to load OAuth configuration: %v", err)
	}
	log.Printf("OAuth configuration loaded")

	// Connect to PostgreSQL
	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("Connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	log.Printf("Repositories initialized")

	// Connect to Redis
	redisURL := cfg.RedisURL
	if strings.HasPrefix(redisURL, "redis://") {
		redisURL = strings.TrimPrefix(redisURL, "redis://")
	}
	// Remove database number if present
	if idx := strings.LastIndex(redisURL, "/"); idx != -1 {
		redisURL = redisURL[:idx]
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	messageCache := cache.NewMessageCache(rdb)
	log.Printf("Connected to Redis")

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	chatService := service.NewChatService(chatRepo)
	messageService := service.NewMessageService(messageRepo, messageCache)
	statusService := service.NewStatusService(rdb)
	profileService := service.NewProfileService(userRepo)
	log.Printf("Services initialized")

	// Initialize handlers
	authHandler := transport.NewAuthHandler(authService)
	profileHandler := transport.NewProfileHandler(profileService)
	chatHandler := transport.NewChatHandler(chatService)
	messageHandler := transport.NewMessageHandler(messageService)
	oauthHandler := httptransport.NewOAuthHandler(oauthCfg, authService)

	// Create router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Auth routes
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.Register).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")
	authRouter.HandleFunc("/google/login", oauthHandler.GoogleLogin).Methods("GET")
	authRouter.HandleFunc("/google/callback", oauthHandler.GoogleCallback).Methods("GET")

	// Protected routes
	chatRouter := router.PathPrefix("/chats").Subrouter()
	chatRouter.Use(middleware.Auth(authService))
	chatRouter.HandleFunc("", chatHandler.CreateChat).Methods("POST")
	chatRouter.HandleFunc("", chatHandler.ListChats).Methods("GET")
	chatRouter.HandleFunc("/{chatId}", chatHandler.GetChat).Methods("GET")
	chatRouter.HandleFunc("/{chatId}/join", chatHandler.JoinChat).Methods("POST")
	chatRouter.HandleFunc("/{chatId}/leave", chatHandler.LeaveChat).Methods("POST")

	messageRouter := router.PathPrefix("/messages").Subrouter()
	messageRouter.Use(middleware.Auth(authService))
	messageRouter.HandleFunc("", messageHandler.Send).Methods("POST")
	messageRouter.HandleFunc("/{messageId}", messageHandler.DeleteMessage).Methods("DELETE")
	messageRouter.HandleFunc("/chat/{chatId}", messageHandler.GetChatHistory).Methods("GET")

	statusRouter := router.PathPrefix("/status").Subrouter()
	statusRouter.Use(middleware.Auth(authService))
	statusRouter.HandleFunc("/online", func(w http.ResponseWriter, r *http.Request) {
		users, err := statusService.GetAllOnlineUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"online_users":` + fmt.Sprintf("%q", users) + `}`))
	}).Methods("GET")

	// WebSocket endpoint
	wsHandler := transport.NewWebSocketHandler(statusService, profileService) // Pass status service
	router.HandleFunc("/ws", wsHandler.HandleWebSocket)
	router.HandleFunc("/api/profile", profileHandler.GetMyProfile).Methods("GET")
	router.HandleFunc("/api/profile", profileHandler.UpdateProfile).Methods("PUT")
	router.HandleFunc("/api/users/{userId}/profile", profileHandler.GetProfile).Methods("GET")
	log.Printf("WebSocket endpoint added")

	// Serve static files from the public directory (must be last)
	staticRouter := router.PathPrefix("/").Subrouter()
	// Serve index.html for root path
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	})
	staticRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("public")))

	// Log all registered routes
	log.Printf("=== Registered Routes ===")
	err = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			pathTemplate = "<no template>"
		}
		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"ANY"}
		}
		log.Printf("Route: %s [%s]", pathTemplate, methods)
		return nil
	})
	if err != nil {
		log.Printf("Error walking routes: %v", err)
	}
	log.Printf("======================")

	// Create server
	srv := &http.Server{
		Addr:    ":8083",
		Handler: router,
	}

	// Start server
	go func() {
		log.Printf("Server is running on port 8083")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server gracefully
	log.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func connectDB(url string) (*gorm.DB, error) {
	log.Printf("Connecting to database...")
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Printf("Connected to database")

	return db, nil
}
