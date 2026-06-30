package cache

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	videoOnlineUserKeyPrefix  = "echo-space:video:online:user:"
	videoOnlineCountKeyPrefix = "echo-space:video:online:count:"
	videoOnlineUserTTL        = 8 * time.Second
	videoOnlineCountTTL       = 10 * time.Second
)

const reportVideoOnlineScript = `
local userKey = KEYS[1]
local countKey = KEYS[2]
local fileId = ARGV[1]
local userTTL = tonumber(ARGV[2])
local countTTL = tonumber(ARGV[3])

if redis.call("EXISTS", userKey) == 0 then
	redis.call("SET", userKey, fileId, "EX", userTTL)
	local count = redis.call("INCR", countKey)
	redis.call("EXPIRE", countKey, countTTL)
	return count
end

redis.call("EXPIRE", userKey, userTTL)
redis.call("EXPIRE", countKey, countTTL)

local count = redis.call("GET", countKey)
if not count then
	return 1
end
return tonumber(count)
`

type VideoOnlineStore struct {
	redis *redis.Client
}

func NewVideoOnlineStore(redisClient *redis.Client) *VideoOnlineStore {
	return &VideoOnlineStore{redis: redisClient}
}

func (s *VideoOnlineStore) Report(ctx context.Context, fileID string, deviceID string) (int, error) {
	if s == nil || s.redis == nil {
		return 1, nil
	}
	fileID = strings.TrimSpace(fileID)
	deviceID = strings.TrimSpace(deviceID)
	if fileID == "" || deviceID == "" {
		return 1, nil
	}

	result, err := s.redis.Eval(ctx, reportVideoOnlineScript, []string{
		videoOnlineUserKey(fileID, deviceID),
		videoOnlineCountKey(fileID),
	}, fileID, int(videoOnlineUserTTL.Seconds()), int(videoOnlineCountTTL.Seconds())).Result()
	if err != nil {
		log.Printf("report video online count redis failed: fileID=%s err=%v", fileID, err)
		return 1, nil
	}

	count, err := parseRedisInt(result)
	if err != nil {
		log.Printf("parse video online count failed: fileID=%s result=%v err=%v", fileID, result, err)
		return 1, nil
	}
	if count <= 0 {
		return 1, nil
	}
	return count, nil
}

func videoOnlineUserKey(fileID string, deviceID string) string {
	return fmt.Sprintf("%s%s:%s", videoOnlineUserKeyPrefix, fileID, deviceID)
}

func videoOnlineCountKey(fileID string) string {
	return videoOnlineCountKeyPrefix + fileID
}
