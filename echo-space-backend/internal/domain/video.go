package domain

import "time"

type VideoInfo struct {
	VideoID            string     `gorm:"column:video_id;primaryKey" json:"videoId"`
	VideoCover         string     `gorm:"column:video_cover" json:"videoCover"`
	VideoName          string     `gorm:"column:video_name" json:"videoName"`
	UserID             string     `gorm:"column:user_id" json:"userId"`
	CreateTime         time.Time  `gorm:"column:create_time" json:"createTime"`
	LastUpdateTime     time.Time  `gorm:"column:last_update_time" json:"lastUpdateTime"`
	PCategoryID        int        `gorm:"column:p_category_id" json:"pCategoryId"`
	CategoryID         *int       `gorm:"column:category_id" json:"categoryId"`
	PostType           int        `gorm:"column:post_type" json:"postType"`
	OriginInfo         *string    `gorm:"column:origin_info" json:"originInfo"`
	Tags               string     `gorm:"column:tags" json:"tags"`
	Introduction       *string    `gorm:"column:introduction" json:"introduction"`
	Interaction        *string    `gorm:"column:interaction" json:"interaction"`
	DownloadPermission int        `gorm:"column:download_permission" json:"downloadPermission"`
	Duration           *int       `gorm:"column:duration" json:"duration"`
	PlayCount          int        `gorm:"column:play_count" json:"playCount"`
	LikeCount          int        `gorm:"column:like_count" json:"likeCount"`
	DanmuCount         int        `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount       int        `gorm:"column:comment_count" json:"commentCount"`
	CoinCount          int        `gorm:"column:coin_count" json:"coinCount"`
	CollectCount       int        `gorm:"column:collect_count" json:"collectCount"`
	RecommendType      int        `gorm:"column:recommend_type" json:"recommendType"`
	LastPlayTime       *time.Time `gorm:"column:last_play_time" json:"lastPlayTime"`
}

func (VideoInfo) TableName() string {
	return "video_info"
}

type VideoInfoFile struct {
	FileID           string `gorm:"column:file_id;primaryKey" json:"fileId"`
	UserID           string `gorm:"column:user_id" json:"userId"`
	VideoID          string `gorm:"column:video_id" json:"videoId"`
	FileName         string `gorm:"column:file_name" json:"fileName"`
	FileIndex        int    `gorm:"column:file_index" json:"fileIndex"`
	FileSize         *int64 `gorm:"column:file_size" json:"fileSize"`
	FilePath         string `gorm:"column:file_path" json:"filePath"`
	Duration         int    `gorm:"column:duration" json:"duration"`
	DownloadStatus   int    `gorm:"column:download_status" json:"downloadStatus"`
	DownloadFilePath string `gorm:"column:download_file_path" json:"downloadFilePath"`
}

func (VideoInfoFile) TableName() string {
	return "video_info_file"
}

type WebVideoItem struct {
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
	PostType           int    `gorm:"column:post_type" json:"postType"`
	OriginInfo         string `gorm:"column:origin_info" json:"originInfo"`
	Tags               string `gorm:"column:tags" json:"tags"`
	Introduction       string `gorm:"column:introduction" json:"introduction"`
	Interaction        string `gorm:"column:interaction" json:"interaction"`
	DownloadPermission int    `gorm:"column:download_permission" json:"downloadPermission"`
	Duration           int    `gorm:"column:duration" json:"duration"`
	PlayTime           string `gorm:"-" json:"playTime"`
	PlayCount          int    `gorm:"column:play_count" json:"playCount"`
	LikeCount          int    `gorm:"column:like_count" json:"likeCount"`
	DanmuCount         int    `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount       int    `gorm:"column:comment_count" json:"commentCount"`
	CoinCount          int    `gorm:"column:coin_count" json:"coinCount"`
	CollectCount       int    `gorm:"column:collect_count" json:"collectCount"`
	RecommendType      int    `gorm:"column:recommend_type" json:"recommendType"`
}

type UcenterAllVideoItem struct {
	VideoID      string `gorm:"column:video_id" json:"videoId"`
	VideoCover   string `gorm:"column:video_cover" json:"videoCover"`
	VideoName    string `gorm:"column:video_name" json:"videoName"`
	CreateTime   string `gorm:"column:create_time" json:"createTime"`
	DanmuCount   int    `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount int    `gorm:"column:comment_count" json:"commentCount"`
}

type VideoSearchDocument struct {
	VideoID      string `gorm:"column:video_id" json:"videoId"`
	UserID       string `gorm:"column:user_id" json:"userId"`
	VideoCover   string `gorm:"column:video_cover" json:"videoCover"`
	VideoName    string `gorm:"column:video_name" json:"videoName"`
	Tags         string `gorm:"column:tags" json:"tags"`
	PlayCount    int    `gorm:"column:play_count" json:"playCount"`
	DanmuCount   int    `gorm:"column:danmu_count" json:"danmuCount"`
	CollectCount int    `gorm:"column:collect_count" json:"collectCount"`
	CreateTime   string `gorm:"column:create_time" json:"createTime"`
}

type DownloadVideoFile struct {
	FileID             string `gorm:"column:file_id"`
	FileName           string `gorm:"column:file_name"`
	VideoID            string `gorm:"column:video_id"`
	VideoName          string `gorm:"column:video_name"`
	DownloadPermission int    `gorm:"column:download_permission"`
	DownloadStatus     int    `gorm:"column:download_status"`
	DownloadFilePath   string `gorm:"column:download_file_path"`
}

type WebVideoDetail struct {
	VideoInfo      WebVideoItem     `json:"videoInfo"`
	UserActionList []UserActionItem `json:"userActionList"`
}
