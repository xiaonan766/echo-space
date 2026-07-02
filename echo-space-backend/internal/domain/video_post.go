package domain

import "time"

const (
	ContentTypeVideo = 0
	ContentTypeImage = 1
)

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
	VideoDownloadStatusNone       = 0
	VideoDownloadStatusGenerating = 1
	VideoDownloadStatusSuccess    = 2
	VideoDownloadStatusFailed     = 3
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
	VideoID            string    `gorm:"column:video_id;primaryKey"`
	VideoCover         string    `gorm:"column:video_cover"`
	VideoName          string    `gorm:"column:video_name"`
	UserID             string    `gorm:"column:user_id"`
	CreateTime         time.Time `gorm:"column:create_time"`
	LastUpdateTime     time.Time `gorm:"column:last_update_time"`
	PCategoryID        int       `gorm:"column:p_category_id"`
	CategoryID         *int      `gorm:"column:category_id"`
	ContentType        int       `gorm:"column:content_type"`
	Status             int       `gorm:"column:status"`
	PostType           int       `gorm:"column:post_type"`
	OriginInfo         *string   `gorm:"column:origin_info"`
	Tags               string    `gorm:"column:tags"`
	Introduction       *string   `gorm:"column:introduction"`
	Interaction        *string   `gorm:"column:interaction"`
	DownloadPermission int       `gorm:"column:download_permission"`
	Duration           *int      `gorm:"column:duration"`
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

type VideoInfoFilePostItem struct {
	FileID         string `gorm:"column:file_id" json:"fileId"`
	UploadID       string `gorm:"column:upload_id" json:"uploadId"`
	UserID         string `gorm:"column:user_id" json:"userId"`
	VideoID        string `gorm:"column:video_id" json:"videoId"`
	FileIndex      int    `gorm:"column:file_index" json:"fileIndex"`
	FileName       string `gorm:"column:file_name" json:"fileName"`
	FileSize       int64  `gorm:"column:file_size" json:"fileSize"`
	FilePath       string `gorm:"column:file_path" json:"filePath"`
	UpdateType     int    `gorm:"column:update_type" json:"updateType"`
	TransferResult int    `gorm:"column:transfer_result" json:"transferResult"`
	Duration       int    `gorm:"column:duration" json:"duration"`
}

type UcenterVideoPostItem struct {
	VideoID            string `gorm:"column:video_id" json:"videoId"`
	VideoCover         string `gorm:"column:video_cover" json:"videoCover"`
	VideoName          string `gorm:"column:video_name" json:"videoName"`
	UserID             string `gorm:"column:user_id" json:"userId"`
	NickName           string `gorm:"column:nick_name" json:"nickName"`
	Avatar             string `gorm:"column:avatar" json:"avatar"`
	CreateTime         string `gorm:"column:create_time" json:"createTime"`
	LastUpdateTime     string `gorm:"column:last_update_time" json:"lastUpdateTime"`
	PCategoryID        int    `gorm:"column:p_category_id" json:"pCategoryId"`
	CategoryID         *int   `gorm:"column:category_id" json:"categoryId"`
	ContentType        int    `gorm:"column:content_type" json:"contentType"`
	Status             int    `gorm:"column:status" json:"status"`
	StatusName         string `gorm:"-" json:"statusName"`
	PostType           int    `gorm:"column:post_type" json:"postType"`
	OriginInfo         string `gorm:"column:origin_info" json:"originInfo"`
	Tags               string `gorm:"column:tags" json:"tags"`
	Introduction       string `gorm:"column:introduction" json:"introduction"`
	Interaction        string `gorm:"column:interaction" json:"interaction"`
	DownloadPermission int    `gorm:"column:download_permission" json:"downloadPermission"`
	Duration           int    `gorm:"column:duration" json:"duration"`
	PlayCount          int    `gorm:"column:play_count" json:"playCount"`
	LikeCount          int    `gorm:"column:like_count" json:"likeCount"`
	DanmuCount         int    `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount       int    `gorm:"column:comment_count" json:"commentCount"`
	CoinCount          int    `gorm:"column:coin_count" json:"coinCount"`
	CollectCount       int    `gorm:"column:collect_count" json:"collectCount"`
}

type AdminVideoPostItem struct {
	VideoID            string `gorm:"column:video_id" json:"videoId"`
	VideoCover         string `gorm:"column:video_cover" json:"videoCover"`
	VideoName          string `gorm:"column:video_name" json:"videoName"`
	UserID             string `gorm:"column:user_id" json:"userId"`
	NickName           string `gorm:"column:nick_name" json:"nickName"`
	Avatar             string `gorm:"column:avatar" json:"avatar"`
	CreateTime         string `gorm:"column:create_time" json:"createTime"`
	LastUpdateTime     string `gorm:"column:last_update_time" json:"lastUpdateTime"`
	PCategoryID        int    `gorm:"column:p_category_id" json:"pCategoryId"`
	CategoryID         *int   `gorm:"column:category_id" json:"categoryId"`
	ContentType        int    `gorm:"column:content_type" json:"contentType"`
	Status             int    `gorm:"column:status" json:"status"`
	StatusName         string `gorm:"-" json:"statusName"`
	PostType           int    `gorm:"column:post_type" json:"postType"`
	OriginInfo         string `gorm:"column:origin_info" json:"originInfo"`
	Tags               string `gorm:"column:tags" json:"tags"`
	Introduction       string `gorm:"column:introduction" json:"introduction"`
	Interaction        string `gorm:"column:interaction" json:"interaction"`
	DownloadPermission int    `gorm:"column:download_permission" json:"downloadPermission"`
	Duration           int    `gorm:"column:duration" json:"duration"`
	PlayCount          int    `gorm:"column:play_count" json:"playCount"`
	LikeCount          int    `gorm:"column:like_count" json:"likeCount"`
	DanmuCount         int    `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount       int    `gorm:"column:comment_count" json:"commentCount"`
	CoinCount          int    `gorm:"column:coin_count" json:"coinCount"`
	CollectCount       int    `gorm:"column:collect_count" json:"collectCount"`
	RecommendType      int    `gorm:"column:recommend_type" json:"recommendType"`
}

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
