package domain

type SysSetting struct {
	RegisterCoinCount  int `json:"registerCoinCount" form:"registerCoinCount"`
	PostVideoCoinCount int `json:"postVideoCoinCount" form:"postVideoCoinCount"`
	VideoSize          int `json:"videoSize" form:"videoSize"`
	VideoPCount        int `json:"videoPCount" form:"videoPCount"`
	VideoCount         int `json:"videoCount" form:"videoCount"`
	CommentCount       int `json:"commentCount" form:"commentCount"`
	DanmuCount         int `json:"danmuCount" form:"danmuCount"`
}

func DefaultSysSetting() SysSetting {
	return SysSetting{
		RegisterCoinCount:  10,
		PostVideoCoinCount: 5,
		VideoSize:          150,
		VideoPCount:        10,
		VideoCount:         10,
		CommentCount:       20,
		DanmuCount:         20,
	}
}
