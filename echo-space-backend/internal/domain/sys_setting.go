package domain

type SysSetting struct {
	RegisterCoinCount         int `json:"registerCoinCount" form:"registerCoinCount"`
	PostVideoCoinCount        int `json:"postVideoCoinCount" form:"postVideoCoinCount"`
	VideoSize                 int `json:"videoSize" form:"videoSize"`
	VideoPCount               int `json:"videoPCount" form:"videoPCount"`
	VideoCount                int `json:"videoCount" form:"videoCount"`
	CommentCount              int `json:"commentCount" form:"commentCount"`
	DanmuCount                int `json:"danmuCount" form:"danmuCount"`
	DanmuUserRateCount        int `json:"danmuUserRateCount" form:"danmuUserRateCount"`
	DanmuUserRateSeconds      int `json:"danmuUserRateSeconds" form:"danmuUserRateSeconds"`
	DanmuUserVideoRateCount   int `json:"danmuUserVideoRateCount" form:"danmuUserVideoRateCount"`
	DanmuUserVideoRateSeconds int `json:"danmuUserVideoRateSeconds" form:"danmuUserVideoRateSeconds"`
	DanmuIPRateCount          int `json:"danmuIPRateCount" form:"danmuIPRateCount"`
	DanmuIPRateSeconds        int `json:"danmuIPRateSeconds" form:"danmuIPRateSeconds"`
	DanmuVideoRateCount       int `json:"danmuVideoRateCount" form:"danmuVideoRateCount"`
	DanmuVideoRateSeconds     int `json:"danmuVideoRateSeconds" form:"danmuVideoRateSeconds"`
}

func DefaultSysSetting() SysSetting {
	return SysSetting{
		RegisterCoinCount:         10,
		PostVideoCoinCount:        5,
		VideoSize:                 150,
		VideoPCount:               10,
		VideoCount:                10,
		CommentCount:              20,
		DanmuCount:                20,
		DanmuUserRateCount:        1,
		DanmuUserRateSeconds:      2,
		DanmuUserVideoRateCount:   5,
		DanmuUserVideoRateSeconds: 60,
		DanmuIPRateCount:          30,
		DanmuIPRateSeconds:        60,
		DanmuVideoRateCount:       300,
		DanmuVideoRateSeconds:     60,
	}
}

func NormalizeSysSetting(setting SysSetting) SysSetting {
	defaultSetting := DefaultSysSetting()
	if setting.RegisterCoinCount <= 0 {
		setting.RegisterCoinCount = defaultSetting.RegisterCoinCount
	}
	if setting.PostVideoCoinCount <= 0 {
		setting.PostVideoCoinCount = defaultSetting.PostVideoCoinCount
	}
	if setting.VideoSize <= 0 {
		setting.VideoSize = defaultSetting.VideoSize
	}
	if setting.VideoPCount <= 0 {
		setting.VideoPCount = defaultSetting.VideoPCount
	}
	if setting.VideoCount <= 0 {
		setting.VideoCount = defaultSetting.VideoCount
	}
	if setting.CommentCount <= 0 {
		setting.CommentCount = defaultSetting.CommentCount
	}
	if setting.DanmuCount <= 0 {
		setting.DanmuCount = defaultSetting.DanmuCount
	}
	if setting.DanmuUserRateCount <= 0 {
		setting.DanmuUserRateCount = defaultSetting.DanmuUserRateCount
	}
	if setting.DanmuUserRateSeconds <= 0 {
		setting.DanmuUserRateSeconds = defaultSetting.DanmuUserRateSeconds
	}
	if setting.DanmuUserVideoRateCount <= 0 {
		setting.DanmuUserVideoRateCount = defaultSetting.DanmuUserVideoRateCount
	}
	if setting.DanmuUserVideoRateSeconds <= 0 {
		setting.DanmuUserVideoRateSeconds = defaultSetting.DanmuUserVideoRateSeconds
	}
	if setting.DanmuIPRateCount <= 0 {
		setting.DanmuIPRateCount = defaultSetting.DanmuIPRateCount
	}
	if setting.DanmuIPRateSeconds <= 0 {
		setting.DanmuIPRateSeconds = defaultSetting.DanmuIPRateSeconds
	}
	if setting.DanmuVideoRateCount <= 0 {
		setting.DanmuVideoRateCount = defaultSetting.DanmuVideoRateCount
	}
	if setting.DanmuVideoRateSeconds <= 0 {
		setting.DanmuVideoRateSeconds = defaultSetting.DanmuVideoRateSeconds
	}
	return setting
}
