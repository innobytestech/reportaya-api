package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration (env-validated, fail-fast).
//
// This is the skeleton configuration: HTTP server, Postgres, Redis, JWT auth,
// CORS, rate limiting, RBAC cache, and the optional audit outbox (Mongo sink).
// Add domain-specific sections as new modules are built.
type Config struct {
	Env      string
	LogLevel string

	HTTP struct {
		Port                    int
		ReadTimeout             time.Duration
		WriteTimeout            time.Duration
		BodyLimitMB             int
		ShutdownTimeout         time.Duration
		ProxyHeader             string
		TrustedProxies          []string
		EnableTrustedProxyCheck bool
		EnableIPValidation      bool
	}

	DB struct {
		DSN             string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
	}

	JWT struct {
		Secret                 string
		Expiration             time.Duration // e.g. 15-30 min
		RefreshExpiration      time.Duration // e.g. 7 days
		Issuer                 string
		AbsoluteSessionTimeout time.Duration // hard cap on a session's lifetime; refresh rotations cannot exceed it
		IdleSessionTimeout     time.Duration // refresh rejected if no authenticated request was seen within this window
		IdleActivityThrottle   time.Duration // minimum interval between activity updates per user (Redis write throttle)
	}

	CORS struct {
		AllowedOrigins []string
		AllowedMethods []string
		AllowedHeaders []string
	}

	Redis struct {
		Host     string
		Port     int
		Username string
		Password string
		DB       int
	}

	RateLimit struct {
		GeneralRPM       int           // requests per minute (global, unauthenticated)
		AuthRPM          int           // requests per minute per authenticated user
		LoginRPM         int           // requests per minute (login endpoint)
		RefreshRPM       int           // requests per minute (refresh endpoint)
		LoginAttempts    int           // max failed attempts before blocking
		LoginBlockWindow time.Duration // duration to block IP after max attempts
	}

	RBAC struct {
		CacheTTL time.Duration
	}

	Mongo struct {
		URI             string
		Database        string
		AuditCollection string
		ConnectTimeout  time.Duration
	}

	Audit struct {
		Enabled           bool
		WorkerBatchSize   int
		WorkerPollEvery   time.Duration
		WorkerMaxAttempts int
		RetentionDays     int
	}

	APIKey   string
	URLFront string
}

// Load loads env (from .env if present), validates and returns Config. Fail-fast on error.
func Load() (*Config, error) {
	_ = godotenv.Load()

	c := &Config{}

	c.Env = getEnv("APP_ENV", "development")
	c.LogLevel = getEnv("LOG_LEVEL", "info")

	strictEnvParsing, err := loadStrictEnvParsing()
	if err != nil {
		return nil, err
	}
	if strictEnvParsing {
		if err := validateEnvFormats(); err != nil {
			return nil, err
		}
	}

	c.loadHTTP()
	if err := c.loadDB(); err != nil {
		return nil, err
	}
	if err := c.loadJWT(); err != nil {
		return nil, err
	}
	if err := c.loadCORS(); err != nil {
		return nil, err
	}
	c.loadRedis()
	c.loadRateLimit()
	c.loadRBAC()
	c.URLFront = strings.TrimRight(getEnv("URL_FRONT", ""), "/")
	if err := c.loadAPIKey(); err != nil {
		return nil, err
	}
	if err := c.validateSensitiveCoreSecrets(); err != nil {
		return nil, err
	}

	c.loadMongo()
	if err := c.loadAudit(); err != nil {
		return nil, err
	}

	return c, nil
}

func loadStrictEnvParsing() (bool, error) {
	strictEnvParsing := true
	if raw, ok := lookupEnvTrimmed("CONFIG_STRICT_PARSING"); ok {
		parsed, err := parseBoolValue(raw)
		if err != nil {
			return false, fmt.Errorf("CONFIG_STRICT_PARSING must be boolean: %w", err)
		}
		strictEnvParsing = parsed
	}
	return strictEnvParsing, nil
}

func (c *Config) loadHTTP() {
	c.HTTP.Port = getEnvInt("HTTP_PORT", 8080)
	c.HTTP.ReadTimeout = getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second)
	c.HTTP.WriteTimeout = getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second)
	c.HTTP.BodyLimitMB = getEnvInt("HTTP_BODY_LIMIT_MB", 2)
	c.HTTP.ShutdownTimeout = getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second)
	c.HTTP.ProxyHeader = getEnv("HTTP_PROXY_HEADER", "X-Forwarded-For")
	c.HTTP.TrustedProxies = getEnvSlice("HTTP_TRUSTED_PROXIES", "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7")
	c.HTTP.EnableTrustedProxyCheck = getEnvBool("HTTP_ENABLE_TRUSTED_PROXY_CHECK", true)
	c.HTTP.EnableIPValidation = getEnvBool("HTTP_ENABLE_IP_VALIDATION", true)
}

func (c *Config) loadDB() error {
	c.DB.DSN = getEnv("DATABASE_URL", "")
	if c.DB.DSN == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	c.DB.MaxOpenConns = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	c.DB.MaxIdleConns = getEnvInt("DB_MAX_IDLE_CONNS", 5)
	c.DB.ConnMaxLifetime = getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	return nil
}

func (c *Config) loadJWT() error {
	c.JWT.Secret = getEnv("JWT_SECRET", "")
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	c.JWT.Expiration = getEnvDuration("JWT_EXPIRATION", 30*time.Minute)
	c.JWT.RefreshExpiration = getEnvDuration("JWT_REFRESH_EXPIRATION", 7*24*time.Hour)
	c.JWT.Issuer = getEnv("JWT_ISSUER", "reportaya-api")
	c.JWT.AbsoluteSessionTimeout = getEnvDuration("JWT_ABSOLUTE_SESSION_TIMEOUT", 4*time.Hour)
	c.JWT.IdleSessionTimeout = getEnvDuration("JWT_IDLE_SESSION_TIMEOUT", 15*time.Minute)
	c.JWT.IdleActivityThrottle = getEnvDuration("JWT_IDLE_ACTIVITY_THROTTLE", 60*time.Second)
	if c.JWT.AbsoluteSessionTimeout <= c.JWT.Expiration {
		return fmt.Errorf("JWT_ABSOLUTE_SESSION_TIMEOUT (%s) must be greater than JWT_EXPIRATION (%s)", c.JWT.AbsoluteSessionTimeout, c.JWT.Expiration)
	}
	if c.JWT.IdleSessionTimeout <= 0 {
		return fmt.Errorf("JWT_IDLE_SESSION_TIMEOUT must be > 0")
	}
	if c.JWT.IdleSessionTimeout >= c.JWT.AbsoluteSessionTimeout {
		return fmt.Errorf("JWT_IDLE_SESSION_TIMEOUT (%s) must be less than JWT_ABSOLUTE_SESSION_TIMEOUT (%s)", c.JWT.IdleSessionTimeout, c.JWT.AbsoluteSessionTimeout)
	}
	if c.JWT.IdleActivityThrottle <= 0 {
		return fmt.Errorf("JWT_IDLE_ACTIVITY_THROTTLE must be > 0")
	}
	if c.JWT.IdleActivityThrottle >= c.JWT.IdleSessionTimeout {
		return fmt.Errorf("JWT_IDLE_ACTIVITY_THROTTLE (%s) must be less than JWT_IDLE_SESSION_TIMEOUT (%s)", c.JWT.IdleActivityThrottle, c.JWT.IdleSessionTimeout)
	}
	return nil
}

func (c *Config) loadCORS() error {
	c.CORS.AllowedOrigins = getEnvSlice("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4200")
	c.CORS.AllowedMethods = getEnvSlice("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	c.CORS.AllowedHeaders = getEnvSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Request-ID,Idempotency-Key")

	// El servidor envía AllowCredentials=true; un origin '*' con credenciales es
	// inseguro (y el navegador lo rechaza). En prod/staging es error de arranque.
	if isSensitiveEnv(c.Env) {
		for _, o := range c.CORS.AllowedOrigins {
			if strings.TrimSpace(o) == "*" {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain '*' in %s (AllowCredentials is enabled)", c.Env)
			}
		}
	}
	return nil
}

func (c *Config) loadRedis() {
	c.Redis.Host = getEnv("REDIS_HOST", "localhost")
	c.Redis.Port = getEnvInt("REDIS_PORT", 6379)
	c.Redis.Username = getEnv("REDIS_USERNAME", "default")
	c.Redis.Password = getEnv("REDIS_PASSWORD", "")
	c.Redis.DB = getEnvInt("REDIS_DB", 0)
}

func (c *Config) loadRateLimit() {
	c.RateLimit.GeneralRPM = getEnvInt("RATE_LIMIT_GENERAL_RPM", 120)
	c.RateLimit.AuthRPM = getEnvInt("RATE_LIMIT_AUTH_RPM", 600)
	c.RateLimit.LoginRPM = getEnvInt("RATE_LIMIT_LOGIN_RPM", 10)
	c.RateLimit.RefreshRPM = getEnvInt("RATE_LIMIT_REFRESH_RPM", 60)
	c.RateLimit.LoginAttempts = getEnvInt("RATE_LIMIT_LOGIN_ATTEMPTS", 5)
	c.RateLimit.LoginBlockWindow = getEnvDuration("RATE_LIMIT_LOGIN_BLOCK_WINDOW", 5*time.Minute)
}

func (c *Config) loadRBAC() {
	c.RBAC.CacheTTL = getEnvDuration("RBAC_CACHE_TTL", 60*time.Second)
}

func (c *Config) loadAPIKey() error {
	c.APIKey = getEnv("API_KEY", "")
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	return nil
}

func (c *Config) validateSensitiveCoreSecrets() error {
	if !isSensitiveEnv(c.Env) {
		return nil
	}
	if err := validateSecret("JWT_SECRET", c.JWT.Secret, 32); err != nil {
		return err
	}
	if err := validateSecret("API_KEY", c.APIKey, 24); err != nil {
		return err
	}
	return nil
}

func (c *Config) loadMongo() {
	c.Mongo.URI = getEnv("MONGO_URI", "")
	c.Mongo.Database = getEnv("MONGO_DB", "")
	if c.Mongo.Database == "" {
		c.Mongo.Database = mongoDatabaseFromURI(c.Mongo.URI)
	}
	if c.Mongo.Database == "" {
		c.Mongo.Database = "reportaya"
	}
	c.Mongo.AuditCollection = getEnv("MONGO_AUDIT_COLLECTION", "audit_logs")
	c.Mongo.ConnectTimeout = getEnvDuration("MONGO_CONNECT_TIMEOUT", 5*time.Second)
}

func (c *Config) loadAudit() error {
	c.Audit.Enabled = getEnvBool("AUDIT_ENABLED", false)
	c.Audit.WorkerBatchSize = getEnvInt("AUDIT_WORKER_BATCH_SIZE", 100)
	c.Audit.WorkerPollEvery = getEnvDuration("AUDIT_WORKER_POLL_EVERY", 2*time.Second)
	c.Audit.WorkerMaxAttempts = getEnvInt("AUDIT_WORKER_MAX_ATTEMPTS", 8)
	c.Audit.RetentionDays = getEnvInt("AUDIT_RETENTION_DAYS", 365)
	if !c.Audit.Enabled {
		return nil
	}
	if c.Mongo.URI == "" {
		return fmt.Errorf("MONGO_URI is required when AUDIT_ENABLED=true")
	}
	if c.Mongo.Database == "" {
		return fmt.Errorf("MONGO_DB is required when AUDIT_ENABLED=true")
	}
	if c.Mongo.AuditCollection == "" {
		return fmt.Errorf("MONGO_AUDIT_COLLECTION is required when AUDIT_ENABLED=true")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

func getEnvSlice(key, defaultCSV string) []string {
	s := os.Getenv(key)
	if s == "" {
		s = defaultCSV
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func getEnvBool(key string, defaultVal bool) bool {
	s := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if s == "" {
		return defaultVal
	}
	v, err := parseBoolValue(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func lookupEnvTrimmed(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func parseBoolValue(raw string) (bool, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return false, fmt.Errorf("empty value")
	}
	switch s {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("got %q", raw)
	}
}

func validateEnvFormats() error {
	invalid := make([]string, 0)

	intKeys := []string{
		"HTTP_PORT",
		"HTTP_BODY_LIMIT_MB",
		"DB_MAX_OPEN_CONNS",
		"DB_MAX_IDLE_CONNS",
		"REDIS_PORT",
		"REDIS_DB",
		"RATE_LIMIT_GENERAL_RPM",
		"RATE_LIMIT_LOGIN_RPM",
		"RATE_LIMIT_LOGIN_ATTEMPTS",
		"AUDIT_WORKER_BATCH_SIZE",
		"AUDIT_WORKER_MAX_ATTEMPTS",
		"AUDIT_RETENTION_DAYS",
	}
	for _, key := range intKeys {
		if raw, ok := lookupEnvTrimmed(key); ok {
			if _, err := strconv.Atoi(raw); err != nil {
				invalid = append(invalid, fmt.Sprintf("%s=%q must be integer", key, raw))
			}
		}
	}

	durationKeys := []string{
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"DB_CONN_MAX_LIFETIME",
		"JWT_EXPIRATION",
		"JWT_REFRESH_EXPIRATION",
		"JWT_ABSOLUTE_SESSION_TIMEOUT",
		"JWT_IDLE_SESSION_TIMEOUT",
		"JWT_IDLE_ACTIVITY_THROTTLE",
		"RATE_LIMIT_LOGIN_BLOCK_WINDOW",
		"RBAC_CACHE_TTL",
		"MONGO_CONNECT_TIMEOUT",
		"AUDIT_WORKER_POLL_EVERY",
	}
	for _, key := range durationKeys {
		if raw, ok := lookupEnvTrimmed(key); ok {
			if _, err := time.ParseDuration(raw); err != nil {
				invalid = append(invalid, fmt.Sprintf("%s=%q must be duration", key, raw))
			}
		}
	}

	boolKeys := []string{
		"CONFIG_STRICT_PARSING",
		"HTTP_ENABLE_TRUSTED_PROXY_CHECK",
		"HTTP_ENABLE_IP_VALIDATION",
		"AUDIT_ENABLED",
	}
	for _, key := range boolKeys {
		if raw, ok := lookupEnvTrimmed(key); ok {
			if _, err := parseBoolValue(raw); err != nil {
				invalid = append(invalid, fmt.Sprintf("%s=%q must be boolean", key, raw))
			}
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("invalid environment configuration: %s", strings.Join(invalid, "; "))
	}

	return nil
}

func isSensitiveEnv(env string) bool {
	v := strings.ToLower(strings.TrimSpace(env))
	return v == "production" || v == "staging"
}

func validateSecret(name, value string, minLen int) error {
	v := strings.TrimSpace(value)
	if len(v) < minLen {
		return fmt.Errorf("%s must be at least %d characters", name, minLen)
	}
	lower := strings.ToLower(v)
	weakMarkers := []string{
		"change-in-production",
		"your-api-key",
		"your-256-bit-secret",
		"changeme",
		"example",
		"default",
		"password",
	}
	for _, marker := range weakMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%s appears to be a placeholder or weak secret", name)
		}
	}
	return nil
}

func mongoDatabaseFromURI(uri string) string {
	v := strings.TrimSpace(uri)
	if v == "" {
		return ""
	}
	u, err := url.Parse(v)
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(strings.TrimPrefix(u.EscapedPath(), "/"))
	if path == "" {
		path = strings.TrimSpace(strings.TrimPrefix(u.Path, "/"))
	}
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	decoded, err := url.PathUnescape(parts[0])
	if err != nil {
		return parts[0]
	}
	return strings.TrimSpace(decoded)
}
