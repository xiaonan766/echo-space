package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
)

type HealthHandler struct {
	redis *redis.Client
}

func NewHealthHandler(redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		redis: redisClient,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	redisStatus := "ok"
	if h.redis == nil {
		redisStatus = "not_configured"
	} else if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		redisStatus = "error"
	}

	response.Success(c, gin.H{
		"module": "admin",
		"redis":  redisStatus,
	})
}
