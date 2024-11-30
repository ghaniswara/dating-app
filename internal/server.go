package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ghaniswara/dating-app/internal/datastore/postgres"
	"github.com/labstack/echo"
	"gorm.io/gorm"
)

type Server struct {
	writer     io.Writer
	httpServer *http.Server
	database   *gorm.DB
}

func NewServer(ctx context.Context, w io.Writer) *Server {
	e := echo.New()

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})

	database, err := postgres.InitializeDB("")

	if err != nil {
		fmt.Fprint(w, "Error initializing database:", err)
		ctx.Err()
	}

	server := &Server{
		httpServer: &http.Server{
			Addr:    ":8080",
			Handler: e,
		},
		database: database,
	}

	server.RegisterRoutes(e)
	return server
}

func (s *Server) RegisterRoutes(e *echo.Echo) {
	e.GET("/healthz", s.handleHealthCheck)
}

func (s *Server) StartServer() error {
	fmt.Fprintf(s.writer, "Server starting on %s\n", s.httpServer.Addr)
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
