package domain

import "time"

type AdminCommentItem struct {
	CommentID     int    `gorm:"column:comment_id" json:"commentId"`
	PCommentID    int    `gorm:"column:p_comment_id" json:"pCommentId"`
	VideoID       string `gorm:"column:video_id" json:"videoId"`
	VideoName     string `gorm:"column:video_name" json:"videoName"`
	VideoCover    string `gorm:"column:video_cover" json:"videoCover"`
	UserID        string `gorm:"column:user_id" json:"userId"`
	NickName      string `gorm:"column:nick_name" json:"nickName"`
	Avatar        string `gorm:"column:avatar" json:"avatar"`
	ReplyUserID   string `gorm:"column:reply_user_id" json:"replyUserId"`
	ReplyNickName string `gorm:"column:reply_nick_name" json:"replyNickName"`
	Content       string `gorm:"column:content" json:"content"`
	ImgPath       string `gorm:"column:img_path" json:"imgPath"`
	PostTime      string `gorm:"column:post_time" json:"postTime"`
}

type AdminDanmuItem struct {
	DanmuID   int     `gorm:"column:danmu_id" json:"danmuId"`
	VideoID   string  `gorm:"column:video_id" json:"videoId"`
	VideoName string  `gorm:"column:video_name" json:"videoName"`
	UserID    string  `gorm:"column:user_id" json:"userId"`
	NickName  string  `gorm:"column:nick_name" json:"nickName"`
	Time      float64 `gorm:"column:time" json:"time"`
	Text      string  `gorm:"column:text" json:"text"`
	PostTime  string  `gorm:"column:post_time" json:"postTime"`
}

type UcenterCommentItem struct {
	CommentID     int    `gorm:"column:comment_id" json:"commentId"`
	PCommentID    int    `gorm:"column:p_comment_id" json:"pCommentId"`
	VideoID       string `gorm:"column:video_id" json:"videoId"`
	VideoName     string `gorm:"column:video_name" json:"videoName"`
	VideoCover    string `gorm:"column:video_cover" json:"videoCover"`
	UserID        string `gorm:"column:user_id" json:"userId"`
	NickName      string `gorm:"column:nick_name" json:"nickName"`
	Avatar        string `gorm:"column:avatar" json:"avatar"`
	ReplyUserID   string `gorm:"column:reply_user_id" json:"replyUserId"`
	ReplyNickName string `gorm:"column:reply_nick_name" json:"replyNickName"`
	Content       string `gorm:"column:content" json:"content"`
	ImgPath       string `gorm:"column:img_path" json:"imgPath"`
	PostTime      string `gorm:"column:post_time" json:"postTime"`
}

type UcenterDanmuItem struct {
	DanmuID    int     `gorm:"column:danmu_id" json:"danmuId"`
	VideoID    string  `gorm:"column:video_id" json:"videoId"`
	VideoName  string  `gorm:"column:video_name" json:"videoName"`
	VideoCover string  `gorm:"column:video_cover" json:"videoCover"`
	UserID     string  `gorm:"column:user_id" json:"userId"`
	NickName   string  `gorm:"column:nick_name" json:"nickName"`
	Time       float64 `gorm:"column:time" json:"time"`
	Text       string  `gorm:"column:text" json:"text"`
	PostTime   string  `gorm:"column:post_time" json:"postTime"`
}

type CommentDeleteInfo struct {
	CommentID  int    `gorm:"column:comment_id"`
	PCommentID int    `gorm:"column:p_comment_id"`
	VideoID    string `gorm:"column:video_id"`
}

type DanmuDeleteInfo struct {
	DanmuID int    `gorm:"column:danmu_id"`
	VideoID string `gorm:"column:video_id"`
}

type VideoComment struct {
	CommentID   int       `gorm:"column:comment_id;primaryKey;autoIncrement" json:"commentId"`
	PCommentID  int       `gorm:"column:p_comment_id" json:"pCommentId"`
	VideoID     string    `gorm:"column:video_id" json:"videoId"`
	VideoUserID string    `gorm:"column:video_user_id" json:"videoUserId"`
	Content     string    `gorm:"column:content" json:"content"`
	ImgPath     string    `gorm:"column:img_path" json:"imgPath"`
	UserID      string    `gorm:"column:user_id" json:"userId"`
	ReplyUserID string    `gorm:"column:reply_user_id" json:"replyUserId"`
	TopType     int       `gorm:"column:top_type" json:"topType"`
	PostTime    time.Time `gorm:"column:post_time" json:"postTime"`
	LikeCount   int       `gorm:"column:like_count" json:"likeCount"`
	HateCount   int       `gorm:"column:hate_count" json:"hateCount"`
}

func (VideoComment) TableName() string {
	return "video_comment"
}

type WebCommentItem struct {
	CommentID     int              `gorm:"column:comment_id" json:"commentId"`
	PCommentID    int              `gorm:"column:p_comment_id" json:"pCommentId"`
	VideoID       string           `gorm:"column:video_id" json:"videoId"`
	VideoUserID   string           `gorm:"column:video_user_id" json:"videoUserId"`
	UserID        string           `gorm:"column:user_id" json:"userId"`
	Avatar        string           `gorm:"column:avatar" json:"avatar"`
	NickName      string           `gorm:"column:nick_name" json:"nickName"`
	ReplyUserID   string           `gorm:"column:reply_user_id" json:"replyUserId"`
	ReplyAvatar   string           `gorm:"column:reply_avatar" json:"replyAvatar"`
	ReplyNickName string           `gorm:"column:reply_nick_name" json:"replyNickName"`
	Content       string           `gorm:"column:content" json:"content"`
	ImgPath       string           `gorm:"column:img_path" json:"imgPath"`
	PostTime      string           `gorm:"column:post_time" json:"postTime"`
	TopType       int              `gorm:"column:top_type" json:"topType"`
	LikeCount     int              `gorm:"column:like_count" json:"likeCount"`
	HateCount     int              `gorm:"column:hate_count" json:"hateCount"`
	ReplyCount    int              `gorm:"-" json:"replyCount"`
	Children      []WebCommentItem `gorm:"-" json:"children"`
}

type CommentCursorPage struct {
	TotalCount int64            `json:"totalCount"`
	PageSize   int              `json:"pageSize"`
	List       []WebCommentItem `json:"list"`
	NextCursor string           `json:"nextCursor"`
	HasMore    bool             `json:"hasMore"`
}

type UserActionItem struct {
	ActionID    int    `gorm:"column:action_id" json:"actionId"`
	VideoID     string `gorm:"column:video_id" json:"videoId"`
	CommentID   int    `gorm:"column:comment_id" json:"commentId"`
	ActionType  int    `gorm:"column:action_type" json:"actionType"`
	ActionCount int    `gorm:"column:action_count" json:"actionCount"`
	UserID      string `gorm:"column:user_id" json:"userId"`
	ActionTime  string `gorm:"column:action_time" json:"actionTime"`
}

type UserAction struct {
	ActionID    int       `gorm:"column:action_id;primaryKey;autoIncrement" json:"actionId"`
	VideoID     string    `gorm:"column:video_id" json:"videoId"`
	VideoUserID string    `gorm:"column:video_user_id" json:"videoUserId"`
	CommentID   int       `gorm:"column:comment_id" json:"commentId"`
	ActionType  int       `gorm:"column:action_type" json:"actionType"`
	ActionCount int       `gorm:"column:action_count" json:"actionCount"`
	UserID      string    `gorm:"column:user_id" json:"userId"`
	ActionTime  time.Time `gorm:"column:action_time" json:"actionTime"`
}

func (UserAction) TableName() string {
	return "user_action"
}

type UserActionVideoTarget struct {
	VideoID string `gorm:"column:video_id"`
	UserID  string `gorm:"column:user_id"`
}

type CommentTargetInfo struct {
	VideoID     string `gorm:"column:video_id"`
	VideoUserID string `gorm:"column:video_user_id"`
	Interaction string `gorm:"column:interaction"`
}

type CommentReplyInfo struct {
	CommentID  int    `gorm:"column:comment_id"`
	PCommentID int    `gorm:"column:p_comment_id"`
	VideoID    string `gorm:"column:video_id"`
	UserID     string `gorm:"column:user_id"`
	NickName   string `gorm:"column:nick_name"`
	Avatar     string `gorm:"column:avatar"`
}

type VideoDanmu struct {
	DanmuID  int       `gorm:"column:danmu_id;primaryKey;autoIncrement" json:"danmuId"`
	VideoID  string    `gorm:"column:video_id" json:"videoId"`
	FileID   string    `gorm:"column:file_id" json:"fileId"`
	UserID   string    `gorm:"column:user_id" json:"userId"`
	PostTime time.Time `gorm:"column:post_time" json:"postTime"`
	Text     string    `gorm:"column:text" json:"text"`
	Mode     int       `gorm:"column:mode" json:"mode"`
	Color    string    `gorm:"column:color" json:"color"`
	Time     int       `gorm:"column:time" json:"time"`
}

func (VideoDanmu) TableName() string {
	return "video_danmu"
}

type WebDanmuItem struct {
	DanmuID  int    `gorm:"column:danmu_id" json:"danmuId"`
	VideoID  string `gorm:"column:video_id" json:"videoId"`
	FileID   string `gorm:"column:file_id" json:"fileId"`
	UserID   string `gorm:"column:user_id" json:"userId"`
	PostTime string `gorm:"column:post_time" json:"postTime"`
	Text     string `gorm:"column:text" json:"text"`
	Mode     int    `gorm:"column:mode" json:"mode"`
	Color    string `gorm:"column:color" json:"color"`
	Time     int    `gorm:"column:time" json:"time"`
}

type DanmuTargetInfo struct {
	VideoID     string `gorm:"column:video_id"`
	FileID      string `gorm:"column:file_id"`
	Interaction string `gorm:"column:interaction"`
}
