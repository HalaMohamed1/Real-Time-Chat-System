package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"rtcs/internal/middleware"
	"rtcs/internal/repository"
	"rtcs/internal/service"
	"rtcs/internal/transport"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestServer struct {
	Router         *mux.Router
	Server         *httptest.Server
	DB             *gorm.DB
	UserRepo       *repository.UserRepository
	ChatRepo       *repository.Repository
	MessageRepo    *repository.MessageRepository
	AuthService    *service.AuthService
	ChatService    *service.ChatService
	ProfileService *service.ProfileService
	MessageService *service.MessageService
}

func NewTestServer() (*TestServer, error) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{}), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GORM: %w", err)
	}

	userRepo := repository.NewUserRepository(gormDB)
	chatRepo := repository.NewChatRepository(gormDB)
	messageRepo := repository.NewMessageRepository(gormDB)

	authService := service.NewAuthService(userRepo, "test-jwt-secret")
	chatService := service.NewChatService(chatRepo)
	profileService := service.NewProfileService(userRepo)
	messageService := service.NewMessageService(messageRepo, nil) // No cache in tests

	router := mux.NewRouter()

	authHandler := transport.NewAuthHandler(authService)
	chatHandler := transport.NewChatHandler(chatService)
	profileHandler := transport.NewProfileHandler(profileService)
	messageHandler := transport.NewMessageHandler(messageService)

	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")

	router.HandleFunc("/api/health", transport.HealthHandler).Methods("GET")

	authMiddleware := middleware.Auth(authService)

	chatRouter := router.PathPrefix("/api/chats").Subrouter()
	chatRouter.Use(authMiddleware)
	chatRouter.HandleFunc("", chatHandler.CreateChat).Methods("POST")
	chatRouter.HandleFunc("", chatHandler.ListChats).Methods("GET")
	chatRouter.HandleFunc("/{chatId}", chatHandler.GetChat).Methods("GET")
	chatRouter.HandleFunc("/{chatId}/join", chatHandler.JoinChat).Methods("POST")
	chatRouter.HandleFunc("/{chatId}/leave", chatHandler.LeaveChat).Methods("POST")

	messageRouter := router.PathPrefix("/api/messages").Subrouter()
	messageRouter.Use(authMiddleware)
	messageRouter.HandleFunc("", messageHandler.Send).Methods("POST")
	messageRouter.HandleFunc("/chat/{chatId}", messageHandler.GetChatHistory).Methods("GET")
	messageRouter.HandleFunc("/{messageId}", messageHandler.DeleteMessage).Methods("DELETE")

	profileRouter := router.PathPrefix("/api/profile").Subrouter()
	profileRouter.Use(authMiddleware)
	profileRouter.HandleFunc("", profileHandler.GetMyProfile).Methods("GET")
	profileRouter.HandleFunc("", profileHandler.UpdateProfile).Methods("PUT")
	profileRouter.HandleFunc("/{userId}", profileHandler.GetProfile).Methods("GET")

	server := httptest.NewServer(router)

	return &TestServer{
		Router:         router,
		Server:         server,
		DB:             gormDB,
		UserRepo:       userRepo,
		MessageRepo:    messageRepo,
		AuthService:    authService,
		ChatService:    chatService,
		ProfileService: profileService,
		MessageService: messageService,
	}, nil
}

func (ts *TestServer) Close() {
	if ts.Server != nil {
		ts.Server.Close()
	}
}

func (ts *TestServer) GetAuthToken(userID string) (string, error) {
	return ts.AuthService.GenerateToken(userID)
}

func (ts *TestServer) ConnectWebSocket(token string) (*websocket.Conn, error) {
	wsURL := fmt.Sprintf("ws%s/ws", ts.Server.URL[4:])

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	header := http.Header{}
	if token != "" {
		header.Add("Authorization", "Bearer "+token)
	}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	return conn, nil
}
