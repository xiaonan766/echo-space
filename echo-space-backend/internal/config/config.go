package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Redis         RedisConfig         `yaml:"redis"`
	MySQL         MySQLConfig         `yaml:"mysql"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	RabbitMQ      RabbitMQConfig      `yaml:"rabbitmq"`
	Admin         AdminConfig         `yaml:"admin"`
	File          FileConfig          `yaml:"file"`
	CommentReview CommentReviewConfig `yaml:"commentReview"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	Mode string `yaml:"mode"`
}

func (c ServerConfig) Address() string {
	if c.Host == "" {
		return ":" + c.Port
	}
	return net.JoinHostPort(c.Host, c.Port)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"poolSize"`
}

type MySQLConfig struct {
	DSN             string `yaml:"dsn"`
	MaxIdleConns    int    `yaml:"maxIdleConns"`
	MaxOpenConns    int    `yaml:"maxOpenConns"`
	ConnMaxLifetime string `yaml:"connMaxLifetime"`
	AutoMigrate     bool   `yaml:"autoMigrate"`
}

type ElasticsearchConfig struct {
	Addresses      []string `yaml:"addresses"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	IndexVideoName string   `yaml:"indexVideoName"`
}

func (c MySQLConfig) ConnMaxLifetimeDuration() time.Duration {
	duration, err := time.ParseDuration(c.ConnMaxLifetime)
	if err != nil {
		return time.Hour
	}
	return duration
}

type RabbitMQConfig struct {
	URL                    string `yaml:"url"`
	CacheRecoveryQueue     string `yaml:"cacheRecoveryQueue"`
	StockLockQueue         string `yaml:"stockLockQueue"`
	VideoTranscodeQueue    string `yaml:"videoTranscodeQueue"`
	DynamicFeedQueue       string `yaml:"dynamicFeedQueue"`
	VideoTranscodePrefetch int    `yaml:"videoTranscodePrefetch"`
	PrefetchCount          int    `yaml:"prefetchCount"`
}

type AdminConfig struct {
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
	TokenTTL string `yaml:"tokenTTL"`
}

func (c AdminConfig) TokenTTLDuration() time.Duration {
	duration, err := time.ParseDuration(c.TokenTTL)
	if err != nil {
		return 24 * time.Hour
	}
	return duration
}

type FileConfig struct {
	ResourceRoot string `yaml:"resourceRoot"`
	MaxImageMB   int    `yaml:"maxImageMB"`
}

type CommentReviewConfig struct {
	Enabled        bool     `yaml:"enabled"`
	RejectMessage  string   `yaml:"rejectMessage"`
	SensitiveWords []string `yaml:"sensitiveWords"`
}

func Load() (Config, error) {
	cfg := defaultConfig()
	if err := loadYAML(&cfg); err != nil {
		return Config{}, err
	}
	applyEnvOverrides(&cfg)
	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "",
			Port: "7070",
			Mode: "debug",
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
		MySQL: MySQLConfig{
			DSN:             "root:root@tcp(localhost:3306)/echo_space?charset=utf8mb4&parseTime=True&loc=Local",
			MaxIdleConns:    5,
			MaxOpenConns:    20,
			ConnMaxLifetime: "1h",
			AutoMigrate:     false,
		},
		Elasticsearch: ElasticsearchConfig{
			Addresses:      []string{"http://localhost:9200"},
			IndexVideoName: "echo_space_video",
		},
		RabbitMQ: RabbitMQConfig{
			URL:                    "amqp://guest:guest@localhost:5672/",
			CacheRecoveryQueue:     "echo-space.shop.cache.recovery",
			StockLockQueue:         "echo-space.shop.stock.lock",
			VideoTranscodeQueue:    "echo-space.video.transcode",
			DynamicFeedQueue:       "echo-space.dynamic.feed",
			VideoTranscodePrefetch: 1,
			PrefetchCount:          20,
		},
		Admin: AdminConfig{
			Account:  "admin",
			Password: "admin123",
			TokenTTL: "24h",
		},
		File: FileConfig{
			ResourceRoot: "resources",
			MaxImageMB:   10,
		},
		CommentReview: CommentReviewConfig{
			Enabled:       true,
			RejectMessage: "评论包含敏感内容，请修改后再发布",
			SensitiveWords: []string{
				"广告", "推广", "引流", "加微信", "微信号", "微信联系", "vx", "v信",
				"加QQ", "QQ群", "联系QQ", "兼职", "刷单", "返利", "代刷", "代充",
				"代付", "低价出售", "博彩", "赌博", "彩票", "赌球", "赌场", "开户",
				"贷款", "套现", "办证", "发票", "裸聊", "约炮", "色情", "成人网站",
				"看片", "外挂", "破解", "盗版资源",
			},
		},
	}
}

func loadYAML(cfg *Config) error {
	path, ok := resolveConfigPath()
	if !ok {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(content, cfg); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

func resolveConfigPath() (string, bool) {
	if path := strings.TrimSpace(os.Getenv("CONFIG_PATH")); path != "" {
		return path, true
	}

	candidates := []string{
		filepath.Join("configs", "application.yaml"),
		filepath.Join("echo-space-backend", "configs", "application.yaml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func applyEnvOverrides(cfg *Config) {
	cfg.Server.Host = envString("SERVER_HOST", cfg.Server.Host)
	cfg.Server.Port = envString("SERVER_PORT", cfg.Server.Port)
	cfg.Server.Mode = envString("GIN_MODE", cfg.Server.Mode)

	cfg.Redis.Addr = envString("REDIS_ADDR", cfg.Redis.Addr)
	cfg.Redis.Password = envString("REDIS_PASSWORD", cfg.Redis.Password)
	cfg.Redis.DB = envInt("REDIS_DB", cfg.Redis.DB)
	cfg.Redis.PoolSize = envInt("REDIS_POOL_SIZE", cfg.Redis.PoolSize)

	cfg.MySQL.DSN = envString("MYSQL_DSN", cfg.MySQL.DSN)
	cfg.MySQL.MaxIdleConns = envInt("MYSQL_MAX_IDLE_CONNS", cfg.MySQL.MaxIdleConns)
	cfg.MySQL.MaxOpenConns = envInt("MYSQL_MAX_OPEN_CONNS", cfg.MySQL.MaxOpenConns)
	cfg.MySQL.ConnMaxLifetime = envString("MYSQL_CONN_MAX_LIFETIME", cfg.MySQL.ConnMaxLifetime)
	cfg.MySQL.AutoMigrate = envBool("MYSQL_AUTO_MIGRATE", cfg.MySQL.AutoMigrate)

	cfg.Elasticsearch.Addresses = envStringSlice("ES_ADDRESSES", cfg.Elasticsearch.Addresses)
	cfg.Elasticsearch.Username = envString("ES_USERNAME", cfg.Elasticsearch.Username)
	cfg.Elasticsearch.Password = envString("ES_PASSWORD", cfg.Elasticsearch.Password)
	cfg.Elasticsearch.IndexVideoName = envString("ES_INDEX_VIDEO_NAME", cfg.Elasticsearch.IndexVideoName)

	cfg.RabbitMQ.URL = envString("RABBITMQ_URL", cfg.RabbitMQ.URL)
	cfg.RabbitMQ.CacheRecoveryQueue = envString("RABBITMQ_CACHE_RECOVERY_QUEUE", cfg.RabbitMQ.CacheRecoveryQueue)
	cfg.RabbitMQ.StockLockQueue = envString("RABBITMQ_STOCK_LOCK_QUEUE", cfg.RabbitMQ.StockLockQueue)
	cfg.RabbitMQ.VideoTranscodeQueue = envString("RABBITMQ_VIDEO_TRANSCODE_QUEUE", cfg.RabbitMQ.VideoTranscodeQueue)
	cfg.RabbitMQ.DynamicFeedQueue = envString("RABBITMQ_DYNAMIC_FEED_QUEUE", cfg.RabbitMQ.DynamicFeedQueue)
	cfg.RabbitMQ.VideoTranscodePrefetch = envInt("RABBITMQ_VIDEO_TRANSCODE_PREFETCH", cfg.RabbitMQ.VideoTranscodePrefetch)
	cfg.RabbitMQ.PrefetchCount = envInt("RABBITMQ_PREFETCH_COUNT", cfg.RabbitMQ.PrefetchCount)

	cfg.Admin.Account = envString("ADMIN_ACCOUNT", cfg.Admin.Account)
	cfg.Admin.Password = envString("ADMIN_PASSWORD", cfg.Admin.Password)
	cfg.Admin.TokenTTL = envString("ADMIN_TOKEN_TTL", cfg.Admin.TokenTTL)

	cfg.File.ResourceRoot = envString("FILE_RESOURCE_ROOT", cfg.File.ResourceRoot)
	cfg.File.MaxImageMB = envInt("FILE_MAX_IMAGE_MB", cfg.File.MaxImageMB)
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envStringSlice(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
