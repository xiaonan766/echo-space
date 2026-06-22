package domain

import "time"

const (
	VideoPostStatusTranscoding    = 0
	VideoPostStatusTransferFailed = 1
	VideoPostStatusPendingReview  = 2
	VideoPostStatusApproved       = 3
	VideoPostStatusRejected       = 4
)

const (
	VideoFileUpdateNone          = 0
	VideoFileUpdateAdded         = 1
	VideoFileUpdateDeletePending = 2
)

const (
	VideoFileTransferProcessing = 0
	VideoFileTransferSuccess    = 1
	VideoFileTransferFailed     = 2
)

const (
	VideoTranscodeMessageWaitPublish = 0
	VideoTranscodeMessagePublished   = 1
	VideoTranscodeMessageProcessing  = 2
	VideoTranscodeMessageSuccess     = 3
	VideoTranscodeMessageRetryWait   = 4
	VideoTranscodeMessageDead        = 5
)

type VideoInfoPost struct {
	VideoID        string    `gorm:"column:video_id;primaryKey"`
	VideoCover     string    `gorm:"column:video_cover"`
	VideoName      string    `gorm:"column:video_name"`
	UserID         string    `gorm:"column:user_id"`
	CreateTime     time.Time `gorm:"column:create_time"`
	LastUpdateTime time.Time `gorm:"column:last_update_time"`
	PCategoryID    int       `gorm:"column:p_category_id"`
	CategoryID     *int      `gorm:"column:category_id"`
	Status         int       `gorm:"column:status"`
	PostType       int       `gorm:"column:post_type"`
	OriginInfo     *string   `gorm:"column:origin_info"`
	Tags           string    `gorm:"column:tags"`
	Introduction   *string   `gorm:"column:introduction"`
	Interaction    *string   `gorm:"column:interaction"`
	Duration       *int      `gorm:"column:duration"`
}

func (VideoInfoPost) TableName() string { return "video_info_post" }

type VideoInfoFilePost struct {
	FileID         string  `gorm:"column:file_id;primaryKey"`
	UploadID       string  `gorm:"column:upload_id"`
	UserID         string  `gorm:"column:user_id"`
	VideoID        string  `gorm:"column:video_id"`
	FileIndex      int     `gorm:"column:file_index"`
	FileName       string  `gorm:"column:file_name"`
	FileSize       *int64  `gorm:"column:file_size"`
	FilePath       *string `gorm:"column:file_path"`
	UpdateType     int     `gorm:"column:update_type"`
	TransferResult int     `gorm:"column:transfer_result"`
	Duration       *int    `gorm:"column:duration"`
}

func (VideoInfoFilePost) TableName() string { return "video_info_file_post" }

type VideoTranscodeMessageRecord struct {
	MessageID     string     `gorm:"column:message_id;primaryKey"`
	FileID        string     `gorm:"column:file_id"`
	VideoID       string     `gorm:"column:video_id"`
	UserID        string     `gorm:"column:user_id"`
	UploadID      string     `gorm:"column:upload_id"`
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

func (VideoTranscodeMessageRecord) TableName() string { return "video_transcode_message" }
