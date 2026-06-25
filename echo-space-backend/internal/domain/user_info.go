package domain

import "time"

type UserInfo struct {
	UserID           string     `gorm:"primaryKey;column:user_id;type:varchar(10)" json:"userId"`
	NickName         string     `gorm:"column:nick_name;type:varchar(20);not null;uniqueIndex:idx_nick_name" json:"nickName"`
	Email            string     `gorm:"column:email;type:varchar(150);not null;uniqueIndex:idx_key_email" json:"email"`
	Password         string     `gorm:"column:password;type:varchar(50);not null" json:"-"`
	Sex              int        `gorm:"column:sex;type:tinyint" json:"sex"`
	Birthday         string     `gorm:"column:birthday;type:varchar(10)" json:"birthday"`
	School           string     `gorm:"column:school;type:varchar(150)" json:"school"`
	PersonIntro      string     `gorm:"column:person_introduction;type:varchar(200)" json:"personIntroduction"`
	JoinTime         time.Time  `gorm:"column:join_time;not null" json:"joinTime"`
	LastLoginTime    *time.Time `gorm:"column:last_login_time" json:"lastLoginTime"`
	LastLoginIP      string     `gorm:"column:last_login_ip;type:varchar(15)" json:"lastLoginIp"`
	Status           int        `gorm:"column:status;type:tinyint;default:1" json:"status"`
	NoticeInfo       string     `gorm:"column:notice_info;type:varchar(300)" json:"noticeInfo"`
	TotalCoinCount   int        `gorm:"column:total_coin_count;not null" json:"totalCoinCount"`
	CurrentCoinCount int        `gorm:"column:current_coin_count;not null" json:"currentCoinCount"`
	Theme            int        `gorm:"column:theme;type:tinyint;not null;default:1" json:"theme"`
	Avatar           string     `gorm:"column:avatar;type:varchar(100);not null;default:''" json:"avatar"`
}

func (UserInfo) TableName() string {
	return "user_info"
}

type UserFocus struct {
	UserID      string    `gorm:"column:user_id;type:varchar(10);primaryKey" json:"userId"`
	FocusUserID string    `gorm:"column:focus_user_id;type:varchar(10);primaryKey" json:"focusUserId"`
	FocusTime   time.Time `gorm:"column:focus_time" json:"focusTime"`
}

func (UserFocus) TableName() string {
	return "user_focus"
}
