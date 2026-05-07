package config

import (
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
}

type ServerConfig struct {
	Host string
	Port string
	Mode string
}

func (c ServerConfig) Address() string {
	if c.Host == "" {
		return ":" + c.Port
	}
	return net.JoinHostPort(c.Host, c.Port)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

func Load() Config {
	return Config{
		Server: ServerConfig{
			Host: envString("SERVER_HOST", ""),
			Port: envString("SERVER_PORT", "7070"),
			Mode: envString("GIN_MODE", "debug"),
		},
		Redis: RedisConfig{
			Addr:     envString("REDIS_ADDR", "localhost:6379"),
			Password: envString("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
			PoolSize: envInt("REDIS_POOL_SIZE", 10),
		},
	}
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
