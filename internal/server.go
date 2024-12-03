package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ghaniswara/dating-app/internal/config"
	"github.com/ghaniswara/dating-app/internal/datastore/postgres"
	matchRepo "github.com/ghaniswara/dating-app/internal/repository/match"
	userRepo "github.com/ghaniswara/dating-app/internal/repository/user"
	routesV1 "github.com/ghaniswara/dating-app/internal/routes/v1"
	authUseCase "github.com/ghaniswara/dating-app/internal/usecase/auth"
	"github.com/ghaniswara/dating-app/internal/usecase/match"
	"github.com/go-redis/redis"
	"github.com/labstack/echo"
	"gorm.io/gorm"
)

func Run(ctx context.Context, w io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	server := NewServer(ctx, w, args[0])

	go func() {
		if err := server.StartServer(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown
	fmt.Fprintf(w, "\nGracefully shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
}

type Server struct {
	httpServer   *http.Server
	database     *gorm.DB
	authUseCase  authUseCase.IAuthUseCase
	matchUseCase match.IMatchUseCase
	userRepo     userRepo.IUserRepo
}

func NewServer(ctx context.Context, w io.Writer, env string) *Server {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	config, err := config.NewConfig(env)

	if err != nil {
		fmt.Fprint(w, "Error loading configurations:", err)
		ctx.Err()
	}

	database, err := postgres.InitializeDB(
		config.Get("POSTGRES_USER"),
		config.Get("POSTGRES_PASSWORD"),
		config.Get("POSTGRES_DB_NAME"),
		config.Get("POSTGRES_HOST"),
		config.Get("POSTGRES_PORT"),
	)

	if err != nil {
		fmt.Fprint(w, "Error initializing database:", err)
		ctx.Err()
	}

	redis := redis.NewClient(&redis.Options{
		Addr: config.Get("REDIS_HOST") + ":" + config.Get("REDIS_PORT"),
	})

	userRepo := userRepo.New(database)
	matchRepo := matchRepo.NewMatchRepo(database, redis)
	authUC := authUseCase.New(userRepo)
	matchUC := match.NewMatchUseCase(
		userRepo,
		redis,
		matchRepo,
	)

	var PORT = config.Get("PORT")

	server := &Server{
		httpServer: &http.Server{
			Addr:    ":" + PORT,
			Handler: e,
		},
		database:     database,
		authUseCase:  authUC,
		matchUseCase: matchUC,
		userRepo:     userRepo,
	}

	server.RegisterRoutes(e)
	return server
}

func (s *Server) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", s.handleHealthCheck)
	routesV1.InitV1Routes(e, s.authUseCase, s.matchUseCase, s.userRepo)
}

func (s *Server) StartServer() error {
	fmt.Printf("Server starting on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
