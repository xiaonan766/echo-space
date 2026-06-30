package domain

type StatisticsInfo struct {
	StatisticsDate  string `gorm:"primaryKey;column:statistics_date;type:varchar(10)" json:"statisticsDate"`
	UserID          string `gorm:"primaryKey;column:user_id;type:varchar(10)" json:"userId"`
	DateType        int    `gorm:"primaryKey;column:date_type;type:tinyint" json:"dateType"`
	StatisticsCount int    `gorm:"column:statistics_count" json:"statisticsCount"`
}

func (StatisticsInfo) TableName() string {
	return "statistics_info"
}

type ActualTimeStatisticsInfo struct {
	PreDayData     map[int]int    `json:"preDayData"`
	TotalCountInfo map[string]int `json:"totalCountInfo"`
}
