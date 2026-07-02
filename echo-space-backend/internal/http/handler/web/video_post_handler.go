package web

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/http/response"
	webservice "github.com/xiaonan766/echo-space/echo-space-backend/internal/service/web"
)

type VideoPostHandler struct {
	service *webservice.VideoPostService
}

func NewVideoPostHandler(service *webservice.VideoPostService) *VideoPostHandler {
	return &VideoPostHandler{service: service}
}

func (h *VideoPostHandler) PostVideo(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}
	pCategoryID, err := strconv.Atoi(strings.TrimSpace(c.PostForm("pCategoryId")))
	if err != nil {
		response.BusinessError(c, "\u8bf7\u9009\u62e9\u6b63\u786e\u7684\u4e00\u7ea7\u5206\u533a", nil)
		return
	}
	postType, err := strconv.Atoi(strings.TrimSpace(c.PostForm("postType")))
	if err != nil {
		response.BusinessError(c, "\u8bf7\u9009\u62e9\u6b63\u786e\u7684\u6295\u7a3f\u7c7b\u578b", nil)
		return
	}
	contentType := 0
	if value := strings.TrimSpace(c.PostForm("contentType")); value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil || (parsed != 0 && parsed != 1) {
			response.BusinessError(c, "请选择正确的稿件类型", nil)
			return
		}
		contentType = parsed
	}
	downloadPermission := 1
	if value := strings.TrimSpace(c.PostForm("downloadPermission")); value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil || (parsed != 0 && parsed != 1) {
			response.BusinessError(c, "请选择正确的视频下载设置", nil)
			return
		}
		downloadPermission = parsed
	}
	var categoryID *int
	if value := strings.TrimSpace(c.PostForm("categoryId")); value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil || parsed <= 0 {
			response.BusinessError(c, "\u8bf7\u9009\u62e9\u6b63\u786e\u7684\u4e8c\u7ea7\u5206\u533a", nil)
			return
		}
		categoryID = &parsed
	}

	var uploadFileList []webservice.VideoPostUploadFile
	if err := json.Unmarshal([]byte(c.PostForm("uploadFileList")), &uploadFileList); err != nil {
		if contentType == 0 {
			response.BusinessError(c, "\u89c6\u9891\u6587\u4ef6\u5217\u8868\u683c\u5f0f\u4e0d\u6b63\u786e", nil)
			return
		}
	}
	var imageList []webservice.ImagePostUploadFile
	if contentType == 1 {
		if err := json.Unmarshal([]byte(c.PostForm("imageList")), &imageList); err != nil {
			response.BusinessError(c, "图片列表格式不正确", nil)
			return
		}
	}
	_, err = h.service.SaveVideoPost(c.Request.Context(), webservice.SaveVideoPostInput{
		UserID: tokenUserInfo.UserID, VideoID: c.PostForm("videoId"),
		VideoCover: c.PostForm("videoCover"), VideoName: c.PostForm("videoName"),
		PCategoryID: pCategoryID, CategoryID: categoryID, ContentType: contentType, PostType: postType,
		OriginInfo: c.PostForm("originInfo"), Tags: c.PostForm("tags"),
		Introduction: c.PostForm("introduction"), Interaction: c.PostForm("interaction"),
		DownloadPermission: downloadPermission,
		UploadFileList:     uploadFileList,
		ImageList:          imageList,
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web post video: %v", err)
		response.ServerError(c, nil)
		return
	}
	response.Success(c, nil)
}

func (h *VideoPostHandler) LoadVideoList(c *gin.Context) {
	tokenUserInfo, ok := getWebTokenUserInfo(c)
	if !ok {
		response.LoginTimeout(c)
		return
	}
	var status *int
	if value := strings.TrimSpace(formOrQuery(c, "status")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			response.BusinessError(c, "\u7a3f\u4ef6\u72b6\u6001\u4e0d\u6b63\u786e", nil)
			return
		}
		status = &parsed
	}
	result, err := h.service.LoadVideoList(c.Request.Context(), webservice.LoadUcenterVideoListInput{
		UserID:         tokenUserInfo.UserID,
		PageNo:         parseWebIntWithDefault(formOrQuery(c, "pageNo"), 1),
		PageSize:       parseWebIntWithDefault(formOrQuery(c, "pageSize"), 15),
		VideoNameFuzzy: formOrQuery(c, "videoNameFuzzy"),
		Status:         status,
		ContentType:    parseOptionalWebInt(formOrQuery(c, "contentType")),
	})
	if err != nil {
		if businessError, ok := webservice.IsBusinessError(err); ok {
			response.BusinessError(c, businessError.Info, nil)
			return
		}
		log.Printf("web load ucenter video list: %v", err)
		response.ServerError(c, nil)
		return
	}
	response.Success(c, result)
}
