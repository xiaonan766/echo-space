package web

import (
	"testing"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func TestCalculateVideoHeatScore(t *testing.T) {
	score := CalculateVideoHeatScore(domain.VideoHotMetrics{
		VideoID:      "Abc123Def4",
		PlayCount:    10,
		LikeCount:    3,
		CollectCount: 4,
		CoinCount:    2,
		CommentCount: 2,
	})
	if score != 73 {
		t.Fatalf("heat score = %d, want %d", score, 73)
	}
}

func TestNormalizeReportVideoPlayHotInput(t *testing.T) {
	input := normalizeReportVideoPlayHotInput(ReportVideoPlayHotInput{
		VideoID:  " Abc123Def4 ",
		DeviceID: " ",
		ClientIP: " 127.0.0.1 ",
	})
	if input.VideoID != "Abc123Def4" {
		t.Fatalf("video id = %q, want %q", input.VideoID, "Abc123Def4")
	}
	if input.DeviceID != "unknown:127.0.0.1" {
		t.Fatalf("device id = %q, want %q", input.DeviceID, "unknown:127.0.0.1")
	}
}

func TestValidateVideoHotMetricEvent(t *testing.T) {
	event := domain.VideoHotMetricEvent{
		EventID:   "event-1",
		VideoID:   "Abc123Def4",
		EventType: domain.VideoHotMetricEventPlay,
		Delta:     1,
	}
	if err := validateVideoHotMetricEvent(event); err != nil {
		t.Fatalf("valid event returned error: %v", err)
	}

	event.EventType = "bad"
	if err := validateVideoHotMetricEvent(event); err == nil {
		t.Fatal("expected unsupported event type error")
	}

	event.EventType = domain.VideoHotMetricEventCollect
	event.Delta = -1
	if err := validateVideoHotMetricEvent(event); err != nil {
		t.Fatalf("collect cancel event returned error: %v", err)
	}

	event.EventType = domain.VideoHotMetricEventCoin
	event.Delta = 0
	if err := validateVideoHotMetricEvent(event); err == nil {
		t.Fatal("expected coin delta error")
	}
}
