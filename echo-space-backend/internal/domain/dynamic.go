package domain

import "time"

const (
	DynamicEventTypeVideo = 1
)

const (
	DynamicFeedMessageWaitPublish = 0
	DynamicFeedMessagePublished   = 1
	DynamicFeedMessageProcessing  = 2
	DynamicFeedMessageSuccess     = 3
	DynamicFeedMessageRetryWait   = 4
	DynamicFeedMessageDead        = 5
)

type DynamicFollowUserItem struct {
	UserID             string `gorm:"column:user_id" json:"userId"`
	NickName           string `gorm:"column:nick_name" json:"nickName"`
	Avatar             string `gorm:"column:avatar" json:"avatar"`
	PersonIntroduction string `gorm:"column:person_introduction" json:"personIntroduction"`
	FocusTime          string `gorm:"column:focus_time" json:"focusTime"`
}

type DynamicCurrentUserInfo struct {
	UserID       string `gorm:"column:user_id" json:"userId"`
	NickName     string `gorm:"column:nick_name" json:"nickName"`
	Avatar       string `gorm:"column:avatar" json:"avatar"`
	FocusCount   int    `gorm:"column:focus_count" json:"focusCount"`
	FansCount    int    `gorm:"column:fans_count" json:"fansCount"`
	DynamicCount int    `gorm:"column:dynamic_count" json:"dynamicCount"`
}

type DynamicFeedPage struct {
	PageSize   int            `json:"pageSize"`
	List       []WebVideoItem `json:"list"`
	NextCursor string         `json:"nextCursor"`
	HasMore    bool           `json:"hasMore"`
}

type DynamicEvent struct {
	EventID      string    `gorm:"column:event_id;primaryKey" json:"eventId"`
	VideoID      string    `gorm:"column:video_id" json:"videoId"`
	AuthorUserID string    `gorm:"column:author_user_id" json:"authorUserId"`
	DynamicTime  time.Time `gorm:"column:dynamic_time" json:"dynamicTime"`
	EventType    int       `gorm:"column:event_type" json:"eventType"`
	CreateTime   time.Time `gorm:"column:create_time" json:"createTime"`
	UpdateTime   time.Time `gorm:"column:update_time" json:"updateTime"`
}

func (DynamicEvent) TableName() string {
	return "dynamic_event"
}

type UserDynamicFeed struct {
	FeedID       string    `gorm:"column:feed_id;primaryKey" json:"feedId"`
	UserID       string    `gorm:"column:user_id" json:"userId"`
	AuthorUserID string    `gorm:"column:author_user_id" json:"authorUserId"`
	VideoID      string    `gorm:"column:video_id" json:"videoId"`
	DynamicTime  time.Time `gorm:"column:dynamic_time" json:"dynamicTime"`
	PushTime     time.Time `gorm:"column:push_time" json:"pushTime"`
	CreateTime   time.Time `gorm:"column:create_time" json:"createTime"`
	UpdateTime   time.Time `gorm:"column:update_time" json:"updateTime"`
}

func (UserDynamicFeed) TableName() string {
	return "user_dynamic_feed"
}

type DynamicFeedMessage struct {
	MessageID    string    `json:"messageId"`
	EventID      string    `json:"eventId"`
	VideoID      string    `json:"videoId"`
	AuthorUserID string    `json:"authorUserId"`
	DynamicTime  time.Time `json:"dynamicTime"`
	EventType    int       `json:"eventType"`
}

type DynamicFeedMessageRecord struct {
	MessageID     string     `gorm:"column:message_id;primaryKey"`
	EventID       string     `gorm:"column:event_id"`
	VideoID       string     `gorm:"column:video_id"`
	AuthorUserID  string     `gorm:"column:author_user_id"`
	MessageStatus int        `gorm:"column:message_status"`
	Payload       string     `gorm:"column:payload"`
	RetryCount    int        `gorm:"column:retry_count"`
	NextRetryTime *time.Time `gorm:"column:next_retry_time"`
	LockedUntil   *time.Time `gorm:"column:locked_until"`
	LockToken     string     `gorm:"column:lock_token"`
	LastError     string     `gorm:"column:last_error"`
	CreateTime    time.Time  `gorm:"column:create_time"`
	UpdateTime    time.Time  `gorm:"column:update_time"`
}

func (DynamicFeedMessageRecord) TableName() string {
	return "dynamic_feed_message"
}
