package domain

import "time"

const (
	VideoHotMetricEventPlay    = "play"
	VideoHotMetricEventLike    = "like"
	VideoHotMetricEventComment = "comment"
)

type VideoHotMetricEvent struct {
	EventID    string    `json:"eventId"`
	VideoID    string    `json:"videoId"`
	EventType  string    `json:"eventType"`
	Delta      int       `json:"delta"`
	OccurredAt time.Time `json:"occurredAt"`
}

type VideoHotMetrics struct {
	VideoID      string `gorm:"column:video_id" json:"videoId"`
	PlayCount    int    `gorm:"column:play_count" json:"playCount"`
	LikeCount    int    `gorm:"column:like_count" json:"likeCount"`
	CommentCount int    `gorm:"column:comment_count" json:"commentCount"`
}

type VideoHotRankEntry struct {
	VideoID   string `json:"videoId"`
	Rank      int    `json:"rank"`
	HeatScore int64  `json:"heatScore"`
}

type WebHotVideoItem struct {
	WebVideoItem
	Rank      int   `json:"rank"`
	HeatScore int64 `json:"heatScore"`
}

type UserActionChange struct {
	VideoID         string
	CommentID       int
	ActionType      int
	VideoCountDelta int
}
