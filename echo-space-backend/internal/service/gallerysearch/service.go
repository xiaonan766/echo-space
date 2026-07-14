package gallerysearch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/embedding"
	searchinfra "github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/search"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

var (
	ErrUnavailable = errors.New("图库搜索服务暂不可用")
	ErrExpired     = errors.New("搜索已过期，请重新搜索")
)

type SearchInput struct {
	SearchType  string
	Keyword     string
	Image       []byte
	SearchToken string
	PageNo      int
	PageSize    int
}

type Service struct {
	repository        *repository.GalleryRepository
	embedder          *embedding.DashScopeClient
	index             *searchinfra.GalleryVectorIndex
	vectorStore       *cache.GallerySearchVectorStore
	resourceRoot      string
	model             string
	minScore          float64
	reconcileInterval time.Duration
	queue             chan string
	pendingMu         sync.Mutex
	pending           map[string]struct{}
}

func NewService(repo *repository.GalleryRepository, embedder *embedding.DashScopeClient, index *searchinfra.GalleryVectorIndex, vectorStore *cache.GallerySearchVectorStore, resourceRoot, model string, minScore float64, reconcileInterval time.Duration) *Service {
	return &Service{repository: repo, embedder: embedder, index: index, vectorStore: vectorStore, resourceRoot: resourceRoot, model: model, minScore: minScore, reconcileInterval: reconcileInterval, queue: make(chan string, 256), pending: make(map[string]struct{})}
}

func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Service) ScheduleSync(imageID string) {
	imageID = strings.TrimSpace(imageID)
	if s == nil || imageID == "" {
		return
	}
	s.pendingMu.Lock()
	if _, exists := s.pending[imageID]; exists {
		s.pendingMu.Unlock()
		return
	}
	s.pending[imageID] = struct{}{}
	s.pendingMu.Unlock()
	select {
	case s.queue <- imageID:
	default:
		s.pendingMu.Lock()
		delete(s.pending, imageID)
		s.pendingMu.Unlock()
		log.Printf("gallery vector sync queue is full: imageID=%s", imageID)
	}
}

func (s *Service) run(ctx context.Context) {
	ticker := time.NewTicker(s.reconcileInterval)
	defer ticker.Stop()
	s.reconcile(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case imageID := <-s.queue:
			if err := s.syncImage(ctx, imageID); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("sync gallery vectors failed: imageID=%s err=%v", imageID, err)
			}
			s.pendingMu.Lock()
			delete(s.pending, imageID)
			s.pendingMu.Unlock()
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

func (s *Service) reconcile(ctx context.Context) {
	imageIDs, err := s.repository.ListApprovedImageIDs(ctx)
	if err != nil {
		log.Printf("reconcile gallery vectors: list approved images: %v", err)
		return
	}
	approved := make(map[string]struct{}, len(imageIDs))
	for _, imageID := range imageIDs {
		approved[imageID] = struct{}{}
		if err := s.syncImage(ctx, imageID); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("reconcile gallery vector failed: imageID=%s err=%v", imageID, err)
		}
	}
	metadata, err := s.index.AllMetadata(ctx)
	if err != nil {
		log.Printf("reconcile gallery vectors: list milvus metadata: %v", err)
		return
	}
	staleFiles := make([]string, 0)
	for _, item := range metadata {
		if _, exists := approved[item.ImageID]; !exists {
			staleFiles = append(staleFiles, item.FileID)
		}
	}
	if err := s.index.DeleteFiles(ctx, staleFiles); err != nil {
		log.Printf("reconcile gallery vectors: delete stale files: %v", err)
	}
}

func (s *Service) syncImage(ctx context.Context, imageID string) error {
	sources, err := s.repository.ListApprovedVectorSourcesByImageID(ctx, imageID)
	if err != nil {
		return err
	}
	current, err := s.index.MetadataByImageID(ctx, imageID)
	if err != nil {
		return err
	}
	currentByFile := make(map[string]searchinfra.GalleryVectorMetadata, len(current))
	for _, item := range current {
		currentByFile[item.FileID] = item
	}
	activeFiles := make(map[string]struct{}, len(sources))
	toEmbed := make([]domain.GalleryVectorSource, 0)
	for _, source := range sources {
		activeFiles[source.FileID] = struct{}{}
		item, exists := currentByFile[source.FileID]
		if !exists || item.ContentVersion != source.ContentVersion || item.EmbeddingModel != s.model {
			toEmbed = append(toEmbed, source)
		}
	}
	for start := 0; start < len(toEmbed); start += 4 {
		end := min(start+4, len(toEmbed))
		batch := toEmbed[start:end]
		images := make([][]byte, 0, len(batch))
		for _, source := range batch {
			path, err := safeResourcePath(s.resourceRoot, source.SourceName)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read gallery source file: %w", err)
			}
			normalized, err := embedding.NormalizeImage(content)
			if err != nil {
				return fmt.Errorf("normalize gallery source image: %w", err)
			}
			images = append(images, normalized)
		}
		vectors, err := s.embedImagesWithRetry(ctx, images)
		if err != nil {
			return err
		}
		records := make([]searchinfra.GalleryVectorRecord, len(batch))
		for offset, source := range batch {
			records[offset] = searchinfra.GalleryVectorRecord{FileID: source.FileID, ImageID: source.ImageID, ContentVersion: source.ContentVersion, EmbeddingModel: s.model, Embedding: vectors[offset]}
		}
		if err := retry(ctx, func() error { return s.index.Upsert(ctx, records) }); err != nil {
			return fmt.Errorf("upsert gallery vectors: %w", err)
		}
	}
	staleFiles := make([]string, 0)
	for _, item := range current {
		if _, exists := activeFiles[item.FileID]; !exists {
			staleFiles = append(staleFiles, item.FileID)
		}
	}
	return retry(ctx, func() error { return s.index.DeleteFiles(ctx, staleFiles) })
}

func (s *Service) Search(ctx context.Context, input SearchInput) (domain.GallerySearchResult, error) {
	input.SearchType = strings.TrimSpace(input.SearchType)
	input.SearchToken = strings.TrimSpace(input.SearchToken)
	input.Keyword = strings.TrimSpace(input.Keyword)
	if input.PageNo <= 0 {
		input.PageNo = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 15
	}
	if input.PageSize > 50 {
		input.PageSize = 50
	}

	var vector []float32
	searchType := input.SearchType
	token := input.SearchToken
	if token != "" {
		cached, exists, err := s.vectorStore.Get(ctx, token)
		if err != nil {
			return domain.GallerySearchResult{}, ErrUnavailable
		}
		if !exists {
			return domain.GallerySearchResult{}, ErrExpired
		}
		vector, searchType = cached.Vector, cached.SearchType
	} else {
		var err error
		switch searchType {
		case "text":
			if input.Keyword == "" || len([]rune(input.Keyword)) > 100 {
				return domain.GallerySearchResult{}, errors.New("搜索关键词长度应为 1–100 个字符")
			}
			vector, err = s.embedder.EmbedText(ctx, input.Keyword)
		case "image":
			var normalized []byte
			normalized, err = embedding.NormalizeImage(input.Image)
			if err == nil {
				var vectors [][]float32
				vectors, err = s.embedder.EmbedImages(ctx, [][]byte{normalized})
				if err == nil {
					vector = vectors[0]
				}
			}
		default:
			return domain.GallerySearchResult{}, errors.New("搜索类型无效")
		}
		if err != nil {
			return domain.GallerySearchResult{}, ErrUnavailable
		}
		token, err = newSearchToken()
		if err != nil {
			return domain.GallerySearchResult{}, ErrUnavailable
		}
		if err := s.vectorStore.Set(ctx, token, cache.GallerySearchVector{SearchType: searchType, Vector: vector}); err != nil {
			return domain.GallerySearchResult{}, ErrUnavailable
		}
	}

	offset := (input.PageNo - 1) * input.PageSize
	hits, err := s.index.Search(ctx, vector, offset, input.PageSize+1, s.minScore)
	if err != nil {
		return domain.GallerySearchResult{}, ErrUnavailable
	}
	items, hasMore, err := s.hydrateHits(ctx, hits, input.PageSize)
	if err != nil {
		return domain.GallerySearchResult{}, ErrUnavailable
	}
	return domain.GallerySearchResult{SearchToken: token, SearchType: searchType, PageNo: input.PageNo, PageSize: input.PageSize, HasMore: hasMore, List: items}, nil
}

func (s *Service) hydrateHits(ctx context.Context, hits []searchinfra.GalleryVectorHit, pageSize int) ([]domain.GallerySearchItem, bool, error) {
	imageIDs := make([]string, 0, len(hits))
	for _, hit := range hits {
		imageIDs = append(imageIDs, hit.ImageID)
	}
	images, err := s.repository.ListApprovedImagesByIDs(ctx, imageIDs)
	if err != nil {
		return nil, false, err
	}
	imageByID := make(map[string]domain.GalleryImageItem, len(images))
	for _, image := range images {
		imageByID[image.ImageID] = image
	}
	items := make([]domain.GallerySearchItem, 0, min(pageSize, len(hits)))
	for _, hit := range hits {
		image, exists := imageByID[hit.ImageID]
		if !exists {
			continue
		}
		matchedImage, approved, err := s.repository.IsApprovedImageFile(ctx, hit.ImageID, hit.FileID)
		if err != nil {
			return nil, false, err
		}
		if !approved {
			continue
		}
		items = append(items, domain.GallerySearchItem{GalleryImageItem: image, MatchedImage: matchedImage, Score: hit.Score})
	}
	sort.SliceStable(items, func(left, right int) bool { return items[left].Score > items[right].Score })
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore, nil
}

func (s *Service) embedImagesWithRetry(ctx context.Context, images [][]byte) ([][]float32, error) {
	var vectors [][]float32
	err := retry(ctx, func() error { var err error; vectors, err = s.embedder.EmbedImages(ctx, images); return err })
	return vectors, err
}

func retry(ctx context.Context, operation func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = operation(); err == nil {
			return nil
		}
		timer := time.NewTimer(time.Duration(1<<attempt) * 300 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func newSearchToken() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func safeResourcePath(root, sourceName string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(rootPath, filepath.FromSlash(strings.TrimLeft(sourceName, "/\\"))))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootPath, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("gallery image path is outside resource root")
	}
	return path, nil
}
