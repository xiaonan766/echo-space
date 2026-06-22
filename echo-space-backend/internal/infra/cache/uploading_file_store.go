package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const uploadingFileKeyPrefix = "echo-space:file:uploading:"

type UploadingFileInfo struct {
	Chunks     int    `json:"chunks"`
	FileName   string `json:"fileName"`
	UploadID   string `json:"uploadId"`
	ChunkIndex int    `json:"chunkIndex"`
	FileSize   int64  `json:"fileSize"`
	FilePath   string `json:"filePath"`
}

type UploadingFileStore struct {
	redis *redis.Client
}

func NewUploadingFileStore(redisClient *redis.Client) *UploadingFileStore {
	return &UploadingFileStore{redis: redisClient}
}

func (s *UploadingFileStore) Create(ctx context.Context, userID string, uploadID string, info UploadingFileInfo, ttl time.Duration) (bool, error) {
	if s == nil || s.redis == nil {
		return false, errors.New("redis is not ready")
	}

	content, err := json.Marshal(info)
	if err != nil {
		return false, err
	}
	return s.redis.SetNX(ctx, uploadingFileKey(userID, uploadID), content, ttl).Result()
}

func (s *UploadingFileStore) Get(ctx context.Context, userID string, uploadID string) (UploadingFileInfo, bool, error) {
	if s == nil || s.redis == nil {
		return UploadingFileInfo{}, false, errors.New("redis is not ready")
	}

	content, err := s.redis.Get(ctx, uploadingFileKey(userID, uploadID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return UploadingFileInfo{}, false, nil
	}
	if err != nil {
		return UploadingFileInfo{}, false, err
	}

	var info UploadingFileInfo
	if err := json.Unmarshal(content, &info); err != nil {
		return UploadingFileInfo{}, false, err
	}
	return info, true, nil
}

func (s *UploadingFileStore) UpdateIfExists(ctx context.Context, userID string, uploadID string, info UploadingFileInfo, ttl time.Duration) (bool, error) {
	if s == nil || s.redis == nil {
		return false, errors.New("redis is not ready")
	}

	content, err := json.Marshal(info)
	if err != nil {
		return false, err
	}
	result, err := updateUploadingFileScript.Run(ctx, s.redis, []string{
		uploadingFileKey(userID, uploadID),
	}, content, int64(ttl/time.Second)).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (s *UploadingFileStore) Delete(ctx context.Context, userID string, uploadID string) error {
	if s == nil || s.redis == nil {
		return errors.New("redis is not ready")
	}
	return s.redis.Del(ctx, uploadingFileKey(userID, uploadID)).Err()
}

func uploadingFileKey(userID string, uploadID string) string {
	return uploadingFileKeyPrefix + userID + ":" + uploadID
}

var updateUploadingFileScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then
	return 0
end
redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2])
return 1
`)
