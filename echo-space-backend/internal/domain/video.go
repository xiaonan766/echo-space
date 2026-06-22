package domain

type WebVideoItem struct {
	VideoID        string `gorm:"column:video_id" json:"videoId"`
	VideoCover     string `gorm:"column:video_cover" json:"videoCover"`
	VideoName      string `gorm:"column:video_name" json:"videoName"`
	UserID         string `gorm:"column:user_id" json:"userId"`
	NickName       string `gorm:"column:nick_name" json:"nickName"`
	Avatar         string `gorm:"column:avatar" json:"avatar"`
	CreateTime     string `gorm:"column:create_time" json:"createTime"`
	LastUpdateTime string `gorm:"column:last_update_time" json:"lastUpdateTime"`
	PCategoryID    int    `gorm:"column:p_category_id" json:"pCategoryId"`
	CategoryID     *int   `gorm:"column:category_id" json:"categoryId"`
	PostType       int    `gorm:"column:post_type" json:"postType"`
	OriginInfo     string `gorm:"column:origin_info" json:"originInfo"`
	Tags           string `gorm:"column:tags" json:"tags"`
	Introduction   string `gorm:"column:introduction" json:"introduction"`
	Interaction    string `gorm:"column:interaction" json:"interaction"`
	Duration       int    `gorm:"column:duration" json:"duration"`
	PlayTime       string `gorm:"-" json:"playTime"`
	PlayCount      int    `gorm:"column:play_count" json:"playCount"`
	LikeCount      int    `gorm:"column:like_count" json:"likeCount"`
	DanmuCount     int    `gorm:"column:danmu_count" json:"danmuCount"`
	CommentCount   int    `gorm:"column:comment_count" json:"commentCount"`
	CoinCount      int    `gorm:"column:coin_count" json:"coinCount"`
	CollectCount   int    `gorm:"column:collect_count" json:"collectCount"`
	RecommendType  int    `gorm:"column:recommend_type" json:"recommendType"`
}
