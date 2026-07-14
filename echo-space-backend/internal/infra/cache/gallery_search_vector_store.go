package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const gallerySearchVectorKeyPrefix = "echo-space:gallery:search-vector:"

type GallerySearchVector struct {
	SearchType string    `json:"searchType"`
	Vector     []float32 `json:"vector"`
}

type GallerySearchVectorStore struct {
	client    *redis.Client
	ttl       time.Duration
	dimension int
}

func NewGallerySearchVectorStore(client *redis.Client, ttl time.Duration, dimension int) *GallerySearchVectorStore {
	return &GallerySearchVectorStore{client: client, ttl: ttl, dimension: dimension}
}

func (s *GallerySearchVectorStore) Set(ctx context.Context, token string, value GallerySearchVector) error {
	if s == nil || s.client == nil || len(value.Vector) != s.dimension {
		return errors.New("gallery search vector store is unavailable")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal gallery search vector: %w", err)
	}
	return s.client.Set(ctx, gallerySearchVectorKeyPrefix+token, payload, s.ttl).Err()
}

func (s *GallerySearchVectorStore) Get(ctx context.Context, token string) (GallerySearchVector, bool, error) {
	if s == nil || s.client == nil {
		return GallerySearchVector{}, false, errors.New("gallery search vector store is unavailable")
	}
	payload, err := s.client.Get(ctx, gallerySearchVectorKeyPrefix+token).Bytes()
	if errors.Is(err, redis.Nil) {
		return GallerySearchVector{}, false, nil
	}
	if err != nil {
		return GallerySearchVector{}, false, err
	}
	var value GallerySearchVector
	if err := json.Unmarshal(payload, &value); err != nil || len(value.Vector) != s.dimension {
		return GallerySearchVector{}, false, errors.New("gallery search vector cache is invalid")
	}
	return value, true, nil
}
