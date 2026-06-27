package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	// Embeds the IANA timezone database in the binary. The prod image is
	// distroless/static (no /usr/share/zoneinfo) and CGO is disabled, so
	// without this time.LoadLocation(...) fails and any scheduler silently
	// falls back to UTC.
	_ "time/tzdata"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"reportaya-api/internal/app/container"
	apphttp "reportaya-api/internal/app/server/http"
	"reportaya-api/internal/config"
	"reportaya-api/internal/observability"
	"reportaya-api/internal/persistence/migrate"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zl := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	logger := &zl

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	if err := migrate.RunTargets([]migrate.Target{
		{
			Name:          "core",
			DatabaseURL:   cfg.DB.DSN,
			MigrationsDir: "file://internal/persistence/migrations",
			Enabled:       true,
		},
	}); err != nil {
		logger.Fatal().Err(err).Msg("failed to run migrations")
	}
	logger.Info().Msg("migrations applied successfully")

	tp, err := observability.InitTracing(context.Background(), "reportaya-api", cfg.Env)
	if err != nil {
		logger.Warn().Err(err).Msg("tracing init failed, continuing without tracing")
	} else {
		defer func() {
			if shutErr := tp.Shutdown(context.Background()); shutErr != nil {
				logger.Warn().Err(shutErr).Msg("tracing shutdown error")
			}
		}()
		if tp.Enabled {
			logger.Info().Msg("opentelemetry tracing enabled")
		}
	}

	ctn, err := container.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize application dependencies")
	}
	defer ctn.Close(context.Background())

	auditCtx, cancelAudit := context.WithCancel(context.Background())
	defer cancelAudit()
	if ctn.AuditWorker != nil {
		go ctn.AuditWorker.Run(auditCtx)
		logger.Info().Msg("audit worker started")
	}

	httpCfg := apphttp.Config{
		Port:                    cfg.HTTP.Port,
		ReadTimeout:             cfg.HTTP.ReadTimeout,
		WriteTimeout:            cfg.HTTP.WriteTimeout,
		BodyLimitMB:             cfg.HTTP.BodyLimitMB,
		ShutdownTimeout:         cfg.HTTP.ShutdownTimeout,
		ProxyHeader:             cfg.HTTP.ProxyHeader,
		TrustedProxies:          cfg.HTTP.TrustedProxies,
		EnableTrustedProxyCheck: cfg.HTTP.EnableTrustedProxyCheck,
		EnableIPValidation:      cfg.HTTP.EnableIPValidation,
		CORSOrigins:             cfg.CORS.AllowedOrigins,
		CORSMethods:             cfg.CORS.AllowedMethods,
		CORSHeaders:             cfg.CORS.AllowedHeaders,
		RateLimitConfig:         cfg,
		RateLimiter:             ctn.RateLimiter,
	}
	srv := apphttp.New(httpCfg, logger, ctn.RegisterRoutes)

	go func() {
		if err := srv.Run(); err != nil {
			logger.Error().Err(err).Msg("server stopped")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("shutdown error")
	}
	cancelAudit()
	logger.Info().Msg("bye")
}
