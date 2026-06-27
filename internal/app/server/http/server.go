package http

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog"

	"reportaya-api/internal/config"
	"reportaya-api/internal/security/ratelimit"
	apperrors "reportaya-api/pkg/errors"
)

// Config for HTTP server.
type Config struct {
	Port                    int
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	BodyLimitMB             int
	ShutdownTimeout         time.Duration
	ProxyHeader             string
	TrustedProxies          []string
	EnableTrustedProxyCheck bool
	EnableIPValidation      bool
	CORSOrigins             []string
	CORSMethods             []string
	CORSHeaders             []string
	RateLimitConfig         *config.Config
	RateLimiter             *ratelimit.RateLimiter
}

// Server wraps Fiber app and provides Run/Shutdown.
type Server struct {
	app  *fiber.App
	cfg  Config
	log  *zerolog.Logger
	addr string
}

// New creates a Fiber app with global middlewares and returns Server.
func New(cfg Config, log *zerolog.Logger, registerRoutes func(*fiber.App)) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:             cfg.ReadTimeout,
		WriteTimeout:            cfg.WriteTimeout,
		BodyLimit:               cfg.BodyLimitMB * 1024 * 1024,
		ProxyHeader:             cfg.ProxyHeader,
		TrustedProxies:          cfg.TrustedProxies,
		EnableTrustedProxyCheck: cfg.EnableTrustedProxyCheck,
		EnableIPValidation:      cfg.EnableIPValidation,
		ErrorHandler: apperrors.FiberErrorHandler(func(path string, err error) {
			log.Error().Err(err).Str("path", path).Msg("request error")
		}),
	})

	// Global middlewares (order matters)
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(TracingMiddleware())
	app.Use(LoggerMiddleware(log))
	app.Use(SecurityHeaders())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORSOrigins, ","),
		AllowMethods:     strings.Join(cfg.CORSMethods, ","),
		AllowHeaders:     strings.Join(cfg.CORSHeaders, ","),
		AllowCredentials: true,
	}))
	// Apply Redis-based rate limiting (global)
	if cfg.RateLimiter != nil && cfg.RateLimitConfig != nil {
		app.Use(RateLimitGeneralMiddleware(cfg.RateLimiter, cfg.RateLimitConfig))
	}

	registerRoutes(app)

	return &Server{
		app:  app,
		cfg:  cfg,
		log:  log,
		addr: fmt.Sprintf(":%d", cfg.Port),
	}
}

// Run starts the server (blocks until Shutdown).
func (s *Server) Run() error {
	return s.app.Listen(s.addr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
