// Package container is the composition root: it builds every infrastructure
// dependency from config and wires them together. Domain services and handlers
// are added here as new modules are built.
package container

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"reportaya-api/internal/audit"
	"reportaya-api/internal/config"
	"reportaya-api/internal/persistence/postgres"
	"reportaya-api/internal/security/jwt"
	"reportaya-api/internal/security/ratelimit"
	"reportaya-api/internal/security/rbac"
	"reportaya-api/internal/security/refresh"
	"reportaya-api/internal/security/sessionactivity"
	"reportaya-api/internal/security/tokenblocklist"
)

// Container holds all application dependencies (composition root).
type Container struct {
	Config *config.Config
	Log    *zerolog.Logger

	DB    *postgres.DB
	Redis *redis.Client

	JWT             *jwt.SignerVerifier
	RefreshJWT      *jwt.SignerVerifier
	RBAC            *rbac.Authorizer
	RateLimiter     *ratelimit.RateLimiter
	RefreshStore    refresh.Store
	TokenBlocklist  tokenblocklist.Store
	SessionActivity sessionactivity.Store

	AuditOutbox  *audit.OutboxRepository
	AuditEmitter *audit.Emitter
	AuditWorker  *audit.Worker
	AuditSink    audit.CloseableSink
}

// New builds the container from config and logger.
func New(cfg *config.Config, log *zerolog.Logger) (ctn *Container, err error) {
	db, err := postgres.New(postgres.Config{
		DSN:             cfg.DB.DSN,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}

	defer func() {
		if err != nil && ctn == nil {
			_ = db.Close()
		}
	}()

	redisClient, err := initRedisClient(cfg, log)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil && ctn == nil {
			_ = redisClient.Close()
		}
	}()

	auditOutbox := audit.NewOutboxRepository(db.DB)

	var auditSink audit.CloseableSink
	var auditWorker *audit.Worker
	if cfg.Audit.Enabled {
		sink, sinkErr := audit.NewMongoSink(
			context.Background(),
			cfg.Mongo.URI,
			cfg.Mongo.Database,
			cfg.Mongo.AuditCollection,
			cfg.Mongo.ConnectTimeout,
			cfg.Audit.RetentionDays,
		)
		if sinkErr != nil {
			return nil, fmt.Errorf("audit sink init failed: %w", sinkErr)
		}
		auditSink = sink
		auditWorker = audit.NewWorker(
			auditOutbox,
			sink,
			log,
			cfg.Audit.WorkerPollEvery,
			cfg.Audit.WorkerBatchSize,
			cfg.Audit.WorkerMaxAttempts,
		)
	}

	ctn = &Container{
		Config:          cfg,
		Log:             log,
		DB:              db,
		Redis:           redisClient,
		JWT:             jwt.NewSignerVerifier(cfg.JWT.Secret, cfg.JWT.Expiration, cfg.JWT.Issuer),
		RefreshJWT:      jwt.NewSignerVerifier(cfg.JWT.Secret, cfg.JWT.RefreshExpiration, cfg.JWT.Issuer),
		RBAC:            rbac.NewAuthorizer(db.DB, redisClient, cfg.RBAC.CacheTTL),
		RateLimiter:     ratelimit.NewRateLimiter(redisClient),
		RefreshStore:    refresh.NewRedisStore(redisClient),
		TokenBlocklist:  tokenblocklist.NewRedisStore(redisClient),
		SessionActivity: sessionactivity.NewRedisStore(redisClient),
		AuditOutbox:     auditOutbox,
		AuditEmitter:    audit.NewEmitter(auditOutbox),
		AuditWorker:     auditWorker,
		AuditSink:       auditSink,
	}
	return ctn, nil
}

func initRedisClient(cfg *config.Config, log *zerolog.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	if log != nil {
		log.Info().Msg("redis connected")
	}
	return client, nil
}

// Close releases all resources held by the container.
func (ctn *Container) Close(ctx context.Context) {
	if ctn == nil {
		return
	}
	if ctn.AuditSink != nil {
		if err := ctn.AuditSink.Close(ctx); err != nil {
			ctn.Log.Warn().Err(err).Msg("audit sink close error")
		}
	}
	if ctn.RBAC != nil {
		ctn.RBAC.Stop()
	}
	if ctn.Redis != nil {
		if err := ctn.Redis.Close(); err != nil {
			ctn.Log.Warn().Err(err).Msg("redis close error")
		}
	}
	if ctn.DB != nil {
		if err := ctn.DB.Close(); err != nil {
			ctn.Log.Warn().Err(err).Msg("postgres close error")
		}
	}
}
