package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
)

type IndexHandler struct{}

func NewIndexHandler() *IndexHandler {
	return &IndexHandler{}
}

func (h *IndexHandler) GetActualTimeStatisticsInfo(c *gin.Context) {
	response.Success(c, gin.H{
		"totalCountInfo": gin.H{
			"userCount":    0,
			"playCount":    0,
			"commentCount": 0,
			"danmuCount":   0,
			"likeCount":    0,
			"collectCount": 0,
			"coinCount":    0,
		},
		"preDayData": gin.H{
			"0": 0,
			"1": 0,
			"2": 0,
			"3": 0,
			"4": 0,
			"5": 0,
			"6": 0,
		},
	})
}

func (h *IndexHandler) GetWeekStatisticsInfo(c *gin.Context) {
	response.Success(c, []gin.H{})
}
