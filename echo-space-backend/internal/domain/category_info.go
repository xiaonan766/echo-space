package domain

type CategoryInfo struct {
	CategoryID   int            `gorm:"primaryKey;autoIncrement;column:category_id" json:"categoryId"`
	PCategoryID  int            `gorm:"column:p_category_id;not null;index:idx_category_parent_sort,priority:1" json:"pCategoryId"`
	CategoryCode string         `gorm:"column:category_code;type:varchar(30);not null;uniqueIndex:uk_category_code" json:"categoryCode"`
	CategoryName string         `gorm:"column:category_name;type:varchar(30);not null" json:"categoryName"`
	Icon         string         `gorm:"column:icon;type:varchar(50)" json:"icon"`
	Background   string         `gorm:"column:background;type:varchar(50)" json:"background"`
	Sort         int            `gorm:"column:sort;type:tinyint;not null;default:0;index:idx_category_parent_sort,priority:2" json:"sort"`
	Children     []CategoryInfo `gorm:"-" json:"children"`
}

func (CategoryInfo) TableName() string {
	return "category_info"
}
