package domain

type DynamicFollowUserItem struct {
	UserID             string `gorm:"column:user_id" json:"userId"`
	NickName           string `gorm:"column:nick_name" json:"nickName"`
	Avatar             string `gorm:"column:avatar" json:"avatar"`
	PersonIntroduction string `gorm:"column:person_introduction" json:"personIntroduction"`
	FocusTime          string `gorm:"column:focus_time" json:"focusTime"`
}

type DynamicFeedPage struct {
	PageSize   int            `json:"pageSize"`
	List       []WebVideoItem `json:"list"`
	NextCursor string         `json:"nextCursor"`
	HasMore    bool           `json:"hasMore"`
}
