package web

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

const (
	defaultHotVideoPageNo     = 1
	defaultHotVideoPageSize   = 20
	maxHotVideoPageSize       = 50
	videoHotBackfillBatchSize = 200
)

type VideoHotMetricPublisher interface {
	PublishVideoHotMetricEvent(ctx context.Context, event domain.VideoHotMetricEvent) error
}

type VideoHotMetricStore interface {
	MarkPlayStart(ctx context.Context, videoID string, deviceID string) (bool, error)
	BeginMetricEvent(ctx context.Context, eventID string) (bool, error)
	ReleaseMetricEvent(ctx context.Context, eventID string) error
	IncrementMetric(ctx context.Context, event domain.VideoHotMetricEvent) (domain.VideoHotMetrics, error)
}

type VideoHotRankStore interface {
	SetMetrics(ctx context.Context, metrics domain.VideoHotMetrics) error
	GetMetrics(ctx context.Context, videoID string) (domain.VideoHotMetrics, error)
	SetRank(ctx context.Context, videoID string, heatScore int64) error
	ListRanks(ctx context.Context, pageNo int, pageSize int) ([]domain.VideoHotRankEntry, int64, error)
}

type VideoHotRepository interface {
	IncrementVideoPlayCount(ctx context.Context, videoID string, delta int) error
	ListVideoHotMetricSnapshots(ctx context.Context, offset int, limit int) ([]domain.VideoHotMetrics, error)
	ListWebVideoByIDs(ctx context.Context, videoIDs []string) ([]domain.WebVideoItem, error)
	ListWebHotVideoByDBPage(ctx context.Context, pageNo int, pageSize int) ([]domain.WebVideoItem, int64, error)
}

type VideoHotMetricService struct {
	repository     VideoHotRepository
	metricStore    VideoHotMetricStore
	rankingService *VideoHotRankingService
	publisher      VideoHotMetricPublisher
	now            func() time.Time
}

type VideoHotRankingService struct {
	repository       VideoHotRepository
	rankStore        VideoHotRankStore
	backfillInterval time.Duration
}

type ReportVideoPlayHotInput struct {
	VideoID  string
	DeviceID string
	ClientIP string
}

type HotVideoListInput struct {
	PageNo   int
	PageSize int
}

func NewVideoHotRankingService(repository VideoHotRepository, rankStore VideoHotRankStore, backfillInterval time.Duration) *VideoHotRankingService {
	if backfillInterval <= 0 {
		backfillInterval = 30 * time.Minute
	}
	return &VideoHotRankingService{
		repository:       repository,
		rankStore:        rankStore,
		backfillInterval: backfillInterval,
	}
}

func NewVideoHotMetricService(repository VideoHotRepository, metricStore VideoHotMetricStore, rankingService *VideoHotRankingService, publisher VideoHotMetricPublisher) *VideoHotMetricService {
	return &VideoHotMetricService{
		repository:     repository,
		metricStore:    metricStore,
		rankingService: rankingService,
		publisher:      publisher,
		now:            time.Now,
	}
}

func (s *VideoHotMetricService) ReportVideoPlayHot(ctx context.Context, input ReportVideoPlayHotInput) error {
	input = normalizeReportVideoPlayHotInput(input)
	if err := validateReportVideoPlayHotInput(input); err != nil {
		return err
	}
	if s == nil {
		return nil
	}

	shouldCount := true
	if s.metricStore != nil {
		counted, err := s.metricStore.MarkPlayStart(ctx, input.VideoID, input.DeviceID)
		if err != nil {
			log.Printf("video hot play dedupe failed, continue as counted: videoID=%s err=%v", input.VideoID, err)
		} else {
			shouldCount = counted
		}
	}
	if !shouldCount {
		return nil
	}

	s.RecordMetric(ctx, input.VideoID, domain.VideoHotMetricEventPlay, 1)
	return nil
}

func (s *VideoHotMetricService) RecordMetric(ctx context.Context, videoID string, eventType string, delta int) {
	if s == nil {
		return
	}
	event, err := s.newMetricEvent(videoID, eventType, delta)
	if err != nil {
		log.Printf("build video hot metric event failed: videoID=%s eventType=%s err=%v", videoID, eventType, err)
		return
	}
	s.publishOrHandleMetricEvent(ctx, event)
}

func (s *VideoHotMetricService) HandleVideoHotMetricEvent(ctx context.Context, event domain.VideoHotMetricEvent) error {
	event = normalizeVideoHotMetricEvent(event)
	if err := validateVideoHotMetricEvent(event); err != nil {
		log.Printf("discard invalid video hot metric event: eventID=%s videoID=%s err=%v", event.EventID, event.VideoID, err)
		return nil
	}
	if s == nil {
		return nil
	}

	metricEventStarted := false
	if s.metricStore != nil {
		started, err := s.metricStore.BeginMetricEvent(ctx, event.EventID)
		if err != nil {
			log.Printf("video hot metric event idempotent mark failed, continue: eventID=%s err=%v", event.EventID, err)
		} else if !started {
			return nil
		} else {
			metricEventStarted = true
		}
	}

	if event.EventType == domain.VideoHotMetricEventPlay && s.repository != nil {
		if err := s.repository.IncrementVideoPlayCount(ctx, event.VideoID, event.Delta); err != nil {
			if metricEventStarted && s.metricStore != nil {
				_ = s.metricStore.ReleaseMetricEvent(ctx, event.EventID)
			}
			return err
		}
	}

	if s.metricStore == nil {
		return nil
	}
	metrics, err := s.metricStore.IncrementMetric(ctx, event)
	if err != nil {
		if metricEventStarted && event.EventType != domain.VideoHotMetricEventPlay {
			_ = s.metricStore.ReleaseMetricEvent(ctx, event.EventID)
			return err
		}
		log.Printf("aggregate video hot metrics failed: eventID=%s videoID=%s err=%v", event.EventID, event.VideoID, err)
		return nil
	}
	if s.rankingService != nil {
		if err := s.rankingService.UpdateVideoRankWithMetrics(ctx, metrics); err != nil {
			log.Printf("update video hot rank failed: videoID=%s err=%v", event.VideoID, err)
		}
	}
	return nil
}

func (s *VideoHotMetricService) publishOrHandleMetricEvent(ctx context.Context, event domain.VideoHotMetricEvent) {
	if s.publisher != nil {
		if err := s.publisher.PublishVideoHotMetricEvent(ctx, event); err == nil {
			return
		} else {
			log.Printf("publish video hot metric event failed, fallback to direct handle: eventID=%s videoID=%s err=%v", event.EventID, event.VideoID, err)
		}
	}
	if err := s.HandleVideoHotMetricEvent(ctx, event); err != nil {
		log.Printf("direct handle video hot metric event failed: eventID=%s videoID=%s err=%v", event.EventID, event.VideoID, err)
	}
}

func (s *VideoHotMetricService) newMetricEvent(videoID string, eventType string, delta int) (domain.VideoHotMetricEvent, error) {
	eventID, err := randomHexToken()
	if err != nil {
		return domain.VideoHotMetricEvent{}, err
	}
	now := time.Now()
	if s != nil && s.now != nil {
		now = s.now()
	}
	event := domain.VideoHotMetricEvent{
		EventID:    eventID,
		VideoID:    strings.TrimSpace(videoID),
		EventType:  strings.TrimSpace(eventType),
		Delta:      delta,
		OccurredAt: now,
	}
	return event, validateVideoHotMetricEvent(event)
}

func (s *VideoHotRankingService) LoadHotVideoList(ctx context.Context, input HotVideoListInput) (domain.PaginationResult[domain.WebHotVideoItem], error) {
	input = normalizeHotVideoListInput(input)
	if s == nil {
		return domain.NewPaginationResult([]domain.WebHotVideoItem{}, 0, input.PageNo, input.PageSize), nil
	}

	if s.rankStore != nil && s.repository != nil {
		ranks, totalCount, err := s.rankStore.ListRanks(ctx, input.PageNo, input.PageSize)
		if err == nil && totalCount > 0 {
			list, err := s.loadHotVideosByRanks(ctx, ranks)
			if err != nil {
				return domain.PaginationResult[domain.WebHotVideoItem]{}, err
			}
			return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
		}
		if err != nil {
			log.Printf("load video hot rank from redis failed, fallback to db: %v", err)
		}
	}

	return s.loadHotVideosFromDB(ctx, input)
}

func (s *VideoHotRankingService) UpdateVideoRank(ctx context.Context, videoID string) error {
	if s == nil || s.rankStore == nil {
		return nil
	}
	metrics, err := s.rankStore.GetMetrics(ctx, videoID)
	if err != nil {
		return err
	}
	return s.UpdateVideoRankWithMetrics(ctx, metrics)
}

func (s *VideoHotRankingService) UpdateVideoRankWithMetrics(ctx context.Context, metrics domain.VideoHotMetrics) error {
	if s == nil || s.rankStore == nil {
		return nil
	}
	metrics = normalizeVideoHotMetrics(metrics)
	if metrics.VideoID == "" {
		return nil
	}
	return s.rankStore.SetRank(ctx, metrics.VideoID, CalculateVideoHeatScore(metrics))
}

func (s *VideoHotRankingService) StartBackfillTasks(ctx context.Context) {
	if s == nil || s.repository == nil || s.rankStore == nil {
		return
	}
	go func() {
		s.backfillAndLog(ctx)
		ticker := time.NewTicker(s.backfillInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.backfillAndLog(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *VideoHotRankingService) RebuildFromDB(ctx context.Context) error {
	if s == nil || s.repository == nil || s.rankStore == nil {
		return nil
	}

	for offset := 0; ; offset += videoHotBackfillBatchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		list, err := s.repository.ListVideoHotMetricSnapshots(ctx, offset, videoHotBackfillBatchSize)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		for _, metrics := range list {
			metrics = normalizeVideoHotMetrics(metrics)
			if err := s.rankStore.SetMetrics(ctx, metrics); err != nil {
				return err
			}
			if err := s.rankStore.SetRank(ctx, metrics.VideoID, CalculateVideoHeatScore(metrics)); err != nil {
				return err
			}
		}
	}
}

func (s *VideoHotRankingService) backfillAndLog(ctx context.Context) {
	if err := s.RebuildFromDB(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("rebuild video hot rank failed: %v", err)
	}
}

func (s *VideoHotRankingService) loadHotVideosByRanks(ctx context.Context, ranks []domain.VideoHotRankEntry) ([]domain.WebHotVideoItem, error) {
	videoIDs := make([]string, 0, len(ranks))
	rankMap := make(map[string]domain.VideoHotRankEntry, len(ranks))
	for _, rank := range ranks {
		videoID := strings.TrimSpace(rank.VideoID)
		if videoID == "" {
			continue
		}
		videoIDs = append(videoIDs, videoID)
		rankMap[videoID] = rank
	}
	if len(videoIDs) == 0 {
		return []domain.WebHotVideoItem{}, nil
	}

	videos, err := s.repository.ListWebVideoByIDs(ctx, videoIDs)
	if err != nil {
		return nil, err
	}
	fillWebVideoPlayTime(videos)

	videoMap := make(map[string]domain.WebVideoItem, len(videos))
	for _, video := range videos {
		videoMap[video.VideoID] = video
	}

	result := make([]domain.WebHotVideoItem, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		video, ok := videoMap[videoID]
		if !ok {
			continue
		}
		rank := rankMap[videoID]
		result = append(result, domain.WebHotVideoItem{
			WebVideoItem: video,
			Rank:         rank.Rank,
			HeatScore:    rank.HeatScore,
		})
	}
	return result, nil
}

func (s *VideoHotRankingService) loadHotVideosFromDB(ctx context.Context, input HotVideoListInput) (domain.PaginationResult[domain.WebHotVideoItem], error) {
	if s == nil || s.repository == nil {
		return domain.NewPaginationResult([]domain.WebHotVideoItem{}, 0, input.PageNo, input.PageSize), nil
	}
	videos, totalCount, err := s.repository.ListWebHotVideoByDBPage(ctx, input.PageNo, input.PageSize)
	if err != nil {
		return domain.PaginationResult[domain.WebHotVideoItem]{}, err
	}
	fillWebVideoPlayTime(videos)

	list := make([]domain.WebHotVideoItem, 0, len(videos))
	startRank := (input.PageNo-1)*input.PageSize + 1
	for index, video := range videos {
		metrics := domain.VideoHotMetrics{
			VideoID:      video.VideoID,
			PlayCount:    video.PlayCount,
			LikeCount:    video.LikeCount,
			CollectCount: video.CollectCount,
			CoinCount:    video.CoinCount,
			CommentCount: video.CommentCount,
		}
		list = append(list, domain.WebHotVideoItem{
			WebVideoItem: video,
			Rank:         startRank + index,
			HeatScore:    CalculateVideoHeatScore(metrics),
		})
	}
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func CalculateVideoHeatScore(metrics domain.VideoHotMetrics) int64 {
	metrics = normalizeVideoHotMetrics(metrics)
	return int64(metrics.PlayCount) +
		int64(metrics.LikeCount)*5 +
		int64(metrics.CollectCount)*5 +
		int64(metrics.CoinCount)*6 +
		int64(metrics.CommentCount)*8
}

func normalizeReportVideoPlayHotInput(input ReportVideoPlayHotInput) ReportVideoPlayHotInput {
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.ClientIP = strings.TrimSpace(input.ClientIP)
	if input.DeviceID == "" {
		if input.ClientIP == "" {
			input.DeviceID = "unknown"
		} else {
			input.DeviceID = "unknown:" + input.ClientIP
		}
	}
	return input
}

func validateReportVideoPlayHotInput(input ReportVideoPlayHotInput) error {
	if len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID) {
		return &BusinessError{Info: "视频ID不正确"}
	}
	if input.DeviceID == "" {
		return errors.New("video hot play device id is empty after normalization")
	}
	return nil
}

func normalizeHotVideoListInput(input HotVideoListInput) HotVideoListInput {
	if input.PageNo <= 0 {
		input.PageNo = defaultHotVideoPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultHotVideoPageSize
	}
	if input.PageSize > maxHotVideoPageSize {
		input.PageSize = maxHotVideoPageSize
	}
	return input
}

func normalizeVideoHotMetricEvent(event domain.VideoHotMetricEvent) domain.VideoHotMetricEvent {
	event.EventID = strings.TrimSpace(event.EventID)
	event.VideoID = strings.TrimSpace(event.VideoID)
	event.EventType = strings.TrimSpace(event.EventType)
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}
	return event
}

func validateVideoHotMetricEvent(event domain.VideoHotMetricEvent) error {
	if event.EventID == "" {
		return errors.New("video hot metric event id is empty")
	}
	if len(event.VideoID) != videoIDLength || !isAlphaNumeric(event.VideoID) {
		return errors.New("video hot metric video id is invalid")
	}
	switch event.EventType {
	case domain.VideoHotMetricEventPlay:
		if event.Delta <= 0 {
			return errors.New("video hot play delta must be positive")
		}
	case domain.VideoHotMetricEventComment:
		if event.Delta <= 0 {
			return errors.New("video hot comment delta must be positive")
		}
	case domain.VideoHotMetricEventLike:
		if event.Delta == 0 {
			return errors.New("video hot like delta must not be zero")
		}
	case domain.VideoHotMetricEventCollect:
		if event.Delta == 0 {
			return errors.New("video hot collect delta must not be zero")
		}
	case domain.VideoHotMetricEventCoin:
		if event.Delta <= 0 {
			return errors.New("video hot coin delta must be positive")
		}
	default:
		return errors.New("video hot metric event type is unsupported")
	}
	return nil
}

func normalizeVideoHotMetrics(metrics domain.VideoHotMetrics) domain.VideoHotMetrics {
	metrics.VideoID = strings.TrimSpace(metrics.VideoID)
	if metrics.PlayCount < 0 {
		metrics.PlayCount = 0
	}
	if metrics.LikeCount < 0 {
		metrics.LikeCount = 0
	}
	if metrics.CollectCount < 0 {
		metrics.CollectCount = 0
	}
	if metrics.CoinCount < 0 {
		metrics.CoinCount = 0
	}
	if metrics.CommentCount < 0 {
		metrics.CommentCount = 0
	}
	return metrics
}
