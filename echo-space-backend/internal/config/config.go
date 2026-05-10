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
	Server ServerConfig `yaml:"server"`
	Redis  RedisConfig  `yaml:"redis"`
	Admin  AdminConfig  `yaml:"admin"`
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
		Admin: AdminConfig{
			Account:  "admin",
			Password: "admin123",
			TokenTTL: "24h",
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

	cfg.Admin.Account = envString("ADMIN_ACCOUNT", cfg.Admin.Account)
	cfg.Admin.Password = envString("ADMIN_PASSWORD", cfg.Admin.Password)
	cfg.Admin.TokenTTL = envString("ADMIN_TOKEN_TTL", cfg.Admin.TokenTTL)
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
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
