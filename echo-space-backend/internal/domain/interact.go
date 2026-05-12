package domain

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

type CommentDeleteInfo struct {
	CommentID  int    `gorm:"column:comment_id"`
	PCommentID int    `gorm:"column:p_comment_id"`
	VideoID    string `gorm:"column:video_id"`
}

type DanmuDeleteInfo struct {
	DanmuID int    `gorm:"column:danmu_id"`
	VideoID string `gorm:"column:video_id"`
}
