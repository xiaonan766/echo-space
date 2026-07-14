package domain

type GalleryImageItem struct {
	ImageID        string `gorm:"column:image_id" json:"imageId"`
	ImageCover     string `gorm:"column:image_cover" json:"imageCover"`
	ImageName      string `gorm:"column:image_name" json:"imageName"`
	UserID         string `gorm:"column:user_id" json:"userId"`
	NickName       string `gorm:"column:nick_name" json:"nickName"`
	Avatar         string `gorm:"column:avatar" json:"avatar"`
	CreateTime     string `gorm:"column:create_time" json:"createTime"`
	LastUpdateTime string `gorm:"column:last_update_time" json:"lastUpdateTime"`
}

type GalleryImageInfo struct {
	ImageID        string `gorm:"column:image_id" json:"imageId"`
	ImageCover     string `gorm:"column:image_cover" json:"imageCover"`
	ImageName      string `gorm:"column:image_name" json:"imageName"`
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
}

type GalleryImageFile struct {
	FileID     string `gorm:"column:file_id" json:"fileId"`
	FileName   string `gorm:"column:file_name" json:"fileName"`
	SourceName string `gorm:"column:source_name" json:"sourceName"`
	FileIndex  int    `gorm:"column:file_index" json:"fileIndex"`
}

type GalleryImageDetail struct {
	ImageInfo GalleryImageInfo   `json:"imageInfo"`
	ImageList []GalleryImageFile `json:"imageList"`
}

type GalleryVectorSource struct {
	FileID         string `gorm:"column:file_id"`
	ImageID        string `gorm:"column:image_id"`
	SourceName     string `gorm:"column:source_name"`
	ContentVersion int64  `gorm:"column:content_version"`
}

type GallerySearchItem struct {
	GalleryImageItem
	MatchedImage string  `json:"matchedImage"`
	Score        float32 `json:"-"`
}

type GallerySearchResult struct {
	SearchToken string              `json:"searchToken"`
	SearchType  string              `json:"searchType"`
	PageNo      int                 `json:"pageNo"`
	PageSize    int                 `json:"pageSize"`
	HasMore     bool                `json:"hasMore"`
	List        []GallerySearchItem `json:"list"`
}
