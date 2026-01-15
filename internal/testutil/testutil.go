package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/vineel/vinagents-go/internal/config"
	"github.com/vineel/vinagents-go/internal/db"
	"github.com/vineel/vinagents-go/internal/handler"
	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/service"
	"github.com/vineel/vinagents-go/pkg/jwt"

	// Import agents to register them
	_ "github.com/vineel/vinagents-go/internal/agent/simple"
)

// TestEnv holds all the test dependencies
type TestEnv struct {
	Config         *config.Config
	Pool           *pgxpool.Pool
	Router         *gin.Engine
	JWTManager     *jwt.Manager
	AuthService    *service.AuthService
	AgentService   *service.AgentService
	ClauserService *service.ClauserService
	UserRepo       *repository.UserRepository
	ClauserRepo    *repository.ClauserRepository
	RiverClient    *river.Client[any]
}

// SetupTestEnv creates a new test environment
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Load test env file
	_ = godotenv.Load("../../.env.test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}

	// Run River migrations
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		t.Fatalf("failed to create river migrator: %v", err)
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		t.Fatalf("failed to run river migrations: %v", err)
	}

	// Create River client
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatalf("failed to create river client: %v", err)
	}

	// Create repositories
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	agentRunRepo := repository.NewAgentRunRepository(pool)
	agentRunMessageRepo := repository.NewAgentRunMessageRepository(pool)
	clauserRepo := repository.NewClauserRepository(pool)
	clauserOutputRepo := repository.NewClauserOutputRepository(pool)

	// Create JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWTSecret,
		cfg.JWTExpiresIn,
		cfg.JWTRefreshSecret,
		cfg.JWTRefreshExpiresIn,
	)

	// Create services
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager)
	agentService := service.NewAgentService(agentRunRepo, agentRunMessageRepo, riverClient, pool)
	clauserService := service.NewClauserService(clauserRepo, clauserOutputRepo, agentRunRepo, riverClient, pool)

	// Create handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo, jwtManager)
	agentHandler := handler.NewAgentHandler(agentService)
	clauserHandler := handler.NewClauserHandler(clauserService)

	// Create router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandler())

	api := router.Group(cfg.APIPrefix)
	{
		healthHandler.RegisterRoutes(api)
		authHandler.RegisterRoutes(api)
		authMiddleware := middleware.Auth(jwtManager)
		userHandler.RegisterRoutes(api, authMiddleware)
		agentHandler.RegisterRoutes(api, authMiddleware)
		clauserHandler.RegisterRoutes(api, authMiddleware)
	}

	env := &TestEnv{
		Config:         cfg,
		Pool:           pool,
		Router:         router,
		JWTManager:     jwtManager,
		AuthService:    authService,
		AgentService:   agentService,
		ClauserService: clauserService,
		UserRepo:       userRepo,
		ClauserRepo:    clauserRepo,
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return env
}

// ResetDatabase truncates all tables in the app schema
func (e *TestEnv) ResetDatabase(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	_, err := e.Pool.Exec(ctx, `
		TRUNCATE TABLE app.clauser_outputs CASCADE;
		TRUNCATE TABLE app.clausers CASCADE;
		TRUNCATE TABLE app.agent_run_messages CASCADE;
		TRUNCATE TABLE app.agent_runs CASCADE;
		TRUNCATE TABLE app.refresh_tokens CASCADE;
		TRUNCATE TABLE app.users CASCADE;
	`)
	if err != nil {
		t.Fatalf("failed to reset database: %v", err)
	}
}

// CreateTestUser creates a user and returns auth tokens
func (e *TestEnv) CreateTestUser(t *testing.T, email, password string) (*service.AuthResponse, error) {
	t.Helper()
	ctx := context.Background()
	return e.AuthService.Register(ctx, email, password, nil, nil)
}

// Request makes a test HTTP request
func (e *TestEnv) Request(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	e.Router.ServeHTTP(w, req)
	return w
}

// ParseResponse parses the JSON response body into the target
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("failed to parse response: %v, body: %s", err, w.Body.String())
	}
}

func init() {
	// Ensure we're using test environment
	if os.Getenv("GO_ENV") == "" {
		os.Setenv("GO_ENV", "test")
	}
}
