package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/cache"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/infra/mq"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/repository"
)

const (
	videoIDLength               = 10
	videoFileIDLength           = 20
	videoMessageIDLength        = 32
	maxVideoCoverLength         = 255
	maxVideoNameLength          = 100
	maxVideoTagsLength          = 300
	maxVideoIntroductionLength  = 2000
	maxVideoOriginInfoLength    = 200
	maxVideoInteractionLength   = 3
	maxVideoPartFileNameLength  = 200
	maxImagePostCount           = 9
	defaultUcenterVideoPageNo   = 1
	defaultUcenterVideoPageSize = 15
	maxUcenterVideoPageSize     = 50
)

var imagePostAllowedExts = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".bmp":  {},
	".webp": {},
}

type VideoTranscodePublisher interface {
	PublishVideoTranscodeMessage(ctx context.Context, message mq.VideoTranscodeMessage) error
}

type VideoPostUploadFile struct {
	UploadID  string `json:"uploadId"`
	FileID    string `json:"fileId"`
	FileName  string `json:"fileName"`
	FileIndex int    `json:"-"`
}

type ImagePostUploadFile struct {
	FileID     string `json:"fileId"`
	SourceName string `json:"sourceName"`
	FileName   string `json:"fileName"`
	FileIndex  int    `json:"-"`
}

type SaveVideoPostInput struct {
	UserID             string
	VideoID            string
	VideoCover         string
	VideoName          string
	PCategoryID        int
	CategoryID         *int
	ContentType        int
	PostType           int
	OriginInfo         string
	Tags               string
	Introduction       string
	Interaction        string
	DownloadPermission int
	UploadFileList     []VideoPostUploadFile
	ImageList          []ImagePostUploadFile
}

type LoadUcenterVideoListInput struct {
	UserID         string
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
	Status         *int
	ContentType    *int
}

type VideoPostService struct {
	repository   *repository.VideoPostRepository
	categoryRepo *repository.CategoryRepository
	uploadStore  *cache.UploadingFileStore
	settingStore VideoUploadSettingStore
	publisher    VideoTranscodePublisher
	resourceRoot string
	now          func() time.Time
	generateID   func(int) (string, error)
}

func NewVideoPostService(
	repo *repository.VideoPostRepository,
	categoryRepo *repository.CategoryRepository,
	uploadStore *cache.UploadingFileStore,
	settingStore VideoUploadSettingStore,
	publisher VideoTranscodePublisher,
	resourceRoot string,
) *VideoPostService {
	resourceRoot = strings.TrimSpace(resourceRoot)
	if resourceRoot == "" {
		resourceRoot = defaultResourceRoot
	}
	return &VideoPostService{
		repository: repo, categoryRepo: categoryRepo, uploadStore: uploadStore,
		settingStore: settingStore, publisher: publisher, resourceRoot: resourceRoot,
		now: time.Now, generateID: randomUploadID,
	}
}

func (s *VideoPostService) SaveVideoPost(ctx context.Context, input SaveVideoPostInput) (string, error) {
	input = normalizeVideoPostInput(input)
	if err := validateVideoPostInput(input); err != nil {
		return "", err
	}
	if s == nil || s.repository == nil || s.categoryRepo == nil {
		return "", errors.New("video post service is not ready")
	}
	if input.ContentType == domain.ContentTypeVideo && s.uploadStore == nil {
		return "", errors.New("video upload store is not ready")
	}
	if err := s.validateCategory(ctx, input.PCategoryID, input.CategoryID); err != nil {
		return "", err
	}
	if input.ContentType == domain.ContentTypeVideo {
		if err := s.validatePartCount(ctx, len(input.UploadFileList)); err != nil {
			return "", err
		}
	} else if err := s.validateImageFiles(input.ImageList); err != nil {
		return "", err
	}

	if input.VideoID == "" {
		if input.ContentType == domain.ContentTypeImage {
			return s.createImagePost(ctx, input)
		}
		return s.createVideoPost(ctx, input)
	}
	if input.ContentType == domain.ContentTypeImage {
		return s.updateImagePost(ctx, input)
	}
	return s.updateVideoPost(ctx, input)
}

func (s *VideoPostService) LoadVideoList(ctx context.Context, input LoadUcenterVideoListInput) (domain.PaginationResult[domain.UcenterVideoPostItem], error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.VideoNameFuzzy = strings.TrimSpace(input.VideoNameFuzzy)
	if input.UserID == "" {
		return domain.PaginationResult[domain.UcenterVideoPostItem]{}, &BusinessError{Info: "\u8bf7\u5148\u767b\u5f55"}
	}
	if utf8.RuneCountInString(input.VideoNameFuzzy) > maxVideoNameLength {
		return domain.PaginationResult[domain.UcenterVideoPostItem]{}, &BusinessError{Info: "\u641c\u7d22\u5173\u952e\u8bcd\u4e0d\u80fd\u8d85\u8fc7100\u4e2a\u5b57\u7b26"}
	}
	if input.Status != nil && (*input.Status < -1 || *input.Status > domain.VideoPostStatusRejected) {
		return domain.PaginationResult[domain.UcenterVideoPostItem]{}, &BusinessError{Info: "\u7a3f\u4ef6\u72b6\u6001\u4e0d\u6b63\u786e"}
	}
	if input.ContentType != nil && !isValidContentType(*input.ContentType) {
		return domain.PaginationResult[domain.UcenterVideoPostItem]{}, &BusinessError{Info: "\u7a3f\u4ef6\u7c7b\u578b\u4e0d\u6b63\u786e"}
	}
	if input.PageNo <= 0 {
		input.PageNo = defaultUcenterVideoPageNo
	}
	if input.PageSize <= 0 {
		input.PageSize = defaultUcenterVideoPageSize
	}
	if input.PageSize > maxUcenterVideoPageSize {
		input.PageSize = maxUcenterVideoPageSize
	}

	list, totalCount, err := s.repository.ListUserPostsByPage(ctx, repository.UcenterVideoPostListQuery{
		UserID: input.UserID, PageNo: input.PageNo, PageSize: input.PageSize,
		VideoNameFuzzy: input.VideoNameFuzzy, Status: input.Status, ContentType: input.ContentType,
	})
	if err != nil {
		return domain.PaginationResult[domain.UcenterVideoPostItem]{}, err
	}
	for index := range list {
		list[index].StatusName = videoPostStatusName(list[index].Status)
	}
	return domain.NewPaginationResult(list, totalCount, input.PageNo, input.PageSize), nil
}

func videoPostStatusName(status int) string {
	switch status {
	case domain.VideoPostStatusTranscoding:
		return "\u8f6c\u7801\u4e2d"
	case domain.VideoPostStatusTransferFailed:
		return "\u8f6c\u7801\u5931\u8d25"
	case domain.VideoPostStatusPendingReview:
		return "\u5f85\u5ba1\u6838"
	case domain.VideoPostStatusApproved:
		return "\u5ba1\u6838\u901a\u8fc7"
	case domain.VideoPostStatusRejected:
		return "\u5ba1\u6838\u672a\u901a\u8fc7"
	default:
		return "\u672a\u77e5\u72b6\u6001"
	}
}

func (s *VideoPostService) createVideoPost(ctx context.Context, input SaveVideoPostInput) (string, error) {
	for _, file := range input.UploadFileList {
		if file.FileID != "" {
			return "", &BusinessError{Info: "\u65b0\u6295\u7a3f\u4e0d\u80fd\u5f15\u7528\u5df2\u6709\u89c6\u9891\u5206P"}
		}
	}
	videoID, err := s.generateID(videoIDLength)
	if err != nil {
		return "", fmt.Errorf("generate video id: %w", err)
	}
	now := s.now()
	post := buildVideoPost(input, videoID, now, domain.VideoPostStatusTranscoding)

	files, messages, publishMessages, err := s.buildNewVideoFiles(ctx, input.UserID, videoID, input.UploadFileList, 1)
	if err != nil {
		return "", err
	}
	if err := s.repository.CreatePost(ctx, repository.SaveNewVideoPostData{
		Post: post, Files: files, Messages: messages,
	}); err != nil {
		return "", fmt.Errorf("create video post: %w", err)
	}
	s.publishAfterCommit(publishMessages)
	return videoID, nil
}

func (s *VideoPostService) createImagePost(ctx context.Context, input SaveVideoPostInput) (string, error) {
	videoID, err := s.generateID(videoIDLength)
	if err != nil {
		return "", fmt.Errorf("generate image post id: %w", err)
	}
	now := s.now()
	post := buildVideoPost(input, videoID, now, domain.VideoPostStatusPendingReview)
	files, err := s.buildImageFiles(input.UserID, videoID, input.ImageList, 1)
	if err != nil {
		return "", err
	}
	if err := s.repository.CreatePost(ctx, repository.SaveNewVideoPostData{
		Post: post, Files: files,
	}); err != nil {
		return "", fmt.Errorf("create image post: %w", err)
	}
	return videoID, nil
}

func (s *VideoPostService) updateVideoPost(ctx context.Context, input SaveVideoPostInput) (string, error) {
	currentPost, currentFiles, err := s.repository.FindPostWithFiles(ctx, input.VideoID, input.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", &BusinessError{Info: "\u89c6\u9891\u4e0d\u5b58\u5728\u6216\u65e0\u6743\u64cd\u4f5c"}
	}
	if err != nil {
		return "", err
	}
	if currentPost.Status == domain.VideoPostStatusTranscoding || currentPost.Status == domain.VideoPostStatusPendingReview {
		return "", &BusinessError{Info: "\u8f6c\u7801\u4e2d\u6216\u5f85\u5ba1\u6838\u7684\u89c6\u9891\u4e0d\u80fd\u4fee\u6539"}
	}
	if currentPost.ContentType != domain.ContentTypeVideo {
		return "", &BusinessError{Info: "\u7a3f\u4ef6\u7c7b\u578b\u4e0d\u5339\u914d"}
	}

	currentByID := make(map[string]domain.VideoInfoFilePost, len(currentFiles))
	for _, file := range currentFiles {
		currentByID[file.FileID] = file
	}
	retained := make(map[string]struct{}, len(input.UploadFileList))
	files := make([]domain.VideoInfoFilePost, 0, len(input.UploadFileList))
	newInputs := make([]VideoPostUploadFile, 0)
	fileNamesChanged := false
	for index, item := range input.UploadFileList {
		if item.FileID == "" {
			item.FileIndex = index + 1
			newInputs = append(newInputs, item)
			continue
		}
		current, ok := currentByID[item.FileID]
		if !ok || current.UpdateType == domain.VideoFileUpdateDeletePending {
			return "", &BusinessError{Info: "\u89c6\u9891\u5206P\u4e0d\u5b58\u5728\u6216\u4e0d\u5c5e\u4e8e\u5f53\u524d\u89c6\u9891"}
		}
		if _, duplicated := retained[item.FileID]; duplicated {
			return "", &BusinessError{Info: "\u89c6\u9891\u5206P\u4e0d\u80fd\u91cd\u590d"}
		}
		retained[item.FileID] = struct{}{}
		if current.FileName != item.FileName {
			fileNamesChanged = true
		}
		current.FileIndex = index + 1
		current.FileName = item.FileName
		files = append(files, current)
	}

	deletedFileIDs := make([]string, 0)
	for _, current := range currentFiles {
		if current.UpdateType == domain.VideoFileUpdateDeletePending {
			continue
		}
		if _, ok := retained[current.FileID]; !ok {
			deletedFileIDs = append(deletedFileIDs, current.FileID)
		}
	}
	newFiles, messages, publishMessages, err := s.buildNewVideoFiles(ctx, input.UserID, input.VideoID, newInputs, 1)
	if err != nil {
		return "", err
	}
	files = append(files, newFiles...)
	if len(files) == 0 {
		return "", &BusinessError{Info: "\u81f3\u5c11\u4fdd\u7559\u4e00\u4e2a\u89c6\u9891\u5206P"}
	}

	status := currentPost.Status
	if len(newFiles) > 0 {
		status = domain.VideoPostStatusTranscoding
	} else if videoPostMetadataChanged(*currentPost, input) || fileNamesChanged || len(deletedFileIDs) > 0 {
		status = domain.VideoPostStatusPendingReview
	}
	post := buildVideoPost(input, input.VideoID, s.now(), status)
	post.CreateTime = currentPost.CreateTime
	post.Duration = currentPost.Duration
	if err := s.repository.UpdatePost(ctx, repository.SaveEditedVideoPostData{
		Post: post, Files: files, DeletedFileIDs: deletedFileIDs, Messages: messages,
	}); errors.Is(err, repository.ErrVideoPostNotEditable) {
		return "", &BusinessError{Info: "\u89c6\u9891\u72b6\u6001\u5df2\u53d8\u66f4\uff0c\u8bf7\u5237\u65b0\u540e\u91cd\u8bd5"}
	} else if err != nil {
		return "", fmt.Errorf("update video post: %w", err)
	}
	s.publishAfterCommit(publishMessages)
	return input.VideoID, nil
}

func (s *VideoPostService) updateImagePost(ctx context.Context, input SaveVideoPostInput) (string, error) {
	currentPost, currentFiles, err := s.repository.FindPostWithFiles(ctx, input.VideoID, input.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", &BusinessError{Info: "\u56fe\u7247\u7a3f\u4ef6\u4e0d\u5b58\u5728\u6216\u65e0\u6743\u64cd\u4f5c"}
	}
	if err != nil {
		return "", err
	}
	if currentPost.ContentType != domain.ContentTypeImage {
		return "", &BusinessError{Info: "\u7a3f\u4ef6\u7c7b\u578b\u4e0d\u5339\u914d"}
	}
	if currentPost.Status == domain.VideoPostStatusTranscoding || currentPost.Status == domain.VideoPostStatusPendingReview {
		return "", &BusinessError{Info: "\u5f85\u5ba1\u6838\u7684\u56fe\u7247\u7a3f\u4ef6\u4e0d\u80fd\u4fee\u6539"}
	}

	deletedFileIDs := make([]string, 0, len(currentFiles))
	for _, current := range currentFiles {
		if current.UpdateType == domain.VideoFileUpdateDeletePending {
			continue
		}
		deletedFileIDs = append(deletedFileIDs, current.FileID)
	}
	files, err := s.buildImageFiles(input.UserID, input.VideoID, input.ImageList, 1)
	if err != nil {
		return "", err
	}
	post := buildVideoPost(input, input.VideoID, s.now(), domain.VideoPostStatusPendingReview)
	post.CreateTime = currentPost.CreateTime
	duration := 0
	post.Duration = &duration
	if err := s.repository.UpdatePost(ctx, repository.SaveEditedVideoPostData{
		Post: post, Files: files, DeletedFileIDs: deletedFileIDs,
	}); errors.Is(err, repository.ErrVideoPostNotEditable) {
		return "", &BusinessError{Info: "\u56fe\u7247\u7a3f\u4ef6\u72b6\u6001\u5df2\u53d8\u66f4\uff0c\u8bf7\u5237\u65b0\u540e\u91cd\u8bd5"}
	} else if err != nil {
		return "", fmt.Errorf("update image post: %w", err)
	}
	return input.VideoID, nil
}

func (s *VideoPostService) buildNewVideoFiles(ctx context.Context, userID string, videoID string, list []VideoPostUploadFile, startIndex int) ([]domain.VideoInfoFilePost, []domain.VideoTranscodeMessageRecord, []mq.VideoTranscodeMessage, error) {
	files := make([]domain.VideoInfoFilePost, 0, len(list))
	records := make([]domain.VideoTranscodeMessageRecord, 0, len(list))
	messages := make([]mq.VideoTranscodeMessage, 0, len(list))
	seenUploadIDs := make(map[string]struct{}, len(list))
	for index, item := range list {
		if !validUploadID(item.UploadID) {
			return nil, nil, nil, &BusinessError{Info: "\u4e0a\u4f20\u4efb\u52a1\u4e0d\u5b58\u5728\uff0c\u8bf7\u91cd\u65b0\u4e0a\u4f20"}
		}
		if _, exists := seenUploadIDs[item.UploadID]; exists {
			return nil, nil, nil, &BusinessError{Info: "\u540c\u4e00\u4e0a\u4f20\u6587\u4ef6\u4e0d\u80fd\u91cd\u590d\u63d0\u4ea4"}
		}
		seenUploadIDs[item.UploadID] = struct{}{}
		info, err := s.validateCompletedUpload(ctx, userID, item.UploadID)
		if err != nil {
			return nil, nil, nil, err
		}
		fileID, err := s.generateID(videoFileIDLength)
		if err != nil {
			return nil, nil, nil, err
		}
		messageID, err := s.generateID(videoMessageIDLength)
		if err != nil {
			return nil, nil, nil, err
		}
		message := mq.VideoTranscodeMessage{
			MessageID: messageID, FileID: fileID, VideoID: videoID, UserID: userID,
			UploadID: item.UploadID, FilePath: info.FilePath, FileName: item.FileName, Chunks: info.Chunks,
		}
		payload, err := json.Marshal(message)
		if err != nil {
			return nil, nil, nil, err
		}
		now := s.now()
		fileIndex := item.FileIndex
		if fileIndex <= 0 {
			fileIndex = startIndex + index
		}
		files = append(files, domain.VideoInfoFilePost{
			FileID: fileID, UploadID: item.UploadID, UserID: userID, VideoID: videoID,
			FileIndex: fileIndex, FileName: item.FileName,
			UpdateType: domain.VideoFileUpdateAdded, TransferResult: domain.VideoFileTransferProcessing,
		})
		records = append(records, domain.VideoTranscodeMessageRecord{
			MessageID: messageID, FileID: fileID, VideoID: videoID, UserID: userID, UploadID: item.UploadID,
			MessageStatus: domain.VideoTranscodeMessageWaitPublish, Payload: string(payload),
			NextRetryTime: &now, CreateTime: now, UpdateTime: now,
		})
		messages = append(messages, message)
	}
	return files, records, messages, nil
}

func (s *VideoPostService) buildImageFiles(userID string, videoID string, list []ImagePostUploadFile, startIndex int) ([]domain.VideoInfoFilePost, error) {
	files := make([]domain.VideoInfoFilePost, 0, len(list))
	for index, item := range list {
		targetPath, err := s.safeImageResourcePath(item.SourceName)
		if err != nil {
			return nil, err
		}
		fileInfo, err := os.Stat(targetPath)
		if err != nil || fileInfo.IsDir() {
			return nil, &BusinessError{Info: "\u56fe\u7247\u6587\u4ef6\u4e0d\u5b58\u5728\uff0c\u8bf7\u91cd\u65b0\u4e0a\u4f20"}
		}
		fileID, err := s.generateID(videoFileIDLength)
		if err != nil {
			return nil, err
		}
		filePath := item.SourceName
		duration := 0
		fileSize := fileInfo.Size()
		fileName := item.FileName
		if fileName == "" {
			fileName = strings.TrimSuffix(filepath.Base(item.SourceName), filepath.Ext(item.SourceName))
		}
		fileIndex := item.FileIndex
		if fileIndex <= 0 {
			fileIndex = startIndex + index
		}
		files = append(files, domain.VideoInfoFilePost{
			FileID: fileID, UserID: userID, VideoID: videoID,
			FileIndex: fileIndex, FileName: fileName, FileSize: &fileSize, FilePath: &filePath,
			UpdateType: domain.VideoFileUpdateAdded, TransferResult: domain.VideoFileTransferSuccess,
			Duration: &duration,
		})
	}
	return files, nil
}

func (s *VideoPostService) validateCompletedUpload(ctx context.Context, userID string, uploadID string) (cache.UploadingFileInfo, error) {
	info, exists, err := s.uploadStore.Get(ctx, userID, uploadID)
	if err != nil {
		return cache.UploadingFileInfo{}, fmt.Errorf("get uploading metadata: %w", err)
	}
	if !exists || info.UploadID != uploadID || info.Chunks <= 0 {
		return cache.UploadingFileInfo{}, &BusinessError{Info: "\u6587\u4ef6\u4e0d\u5b58\u5728\uff0c\u8bf7\u91cd\u65b0\u4e0a\u4f20"}
	}
	uploadDir, err := safeResourceSubdirectory(filepath.Join(s.resourceRoot, temporaryVideoDir), info.FilePath)
	if err != nil {
		return cache.UploadingFileInfo{}, fmt.Errorf("resolve uploading directory: %w", err)
	}
	state, err := scanUploadDiskState(uploadDir, info.Chunks)
	if errors.Is(err, os.ErrNotExist) {
		return cache.UploadingFileInfo{}, &BusinessError{Info: "\u6587\u4ef6\u4e0d\u5b58\u5728\uff0c\u8bf7\u91cd\u65b0\u4e0a\u4f20"}
	}
	if err != nil {
		return cache.UploadingFileInfo{}, err
	}
	if state.HighestContiguous != info.Chunks-1 || len(state.ChunkSizes) != info.Chunks || state.TotalSize <= 0 {
		return cache.UploadingFileInfo{}, &BusinessError{Info: "\u89c6\u9891\u5206\u7247\u5c1a\u672a\u4e0a\u4f20\u5b8c\u6210"}
	}
	return info, nil
}

func (s *VideoPostService) validateCategory(ctx context.Context, parentID int, categoryID *int) error {
	parent, err := s.categoryRepo.FindByID(ctx, parentID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && parent.PCategoryID != 0) {
		return &BusinessError{Info: "\u4e00\u7ea7\u5206\u533a\u4e0d\u5b58\u5728"}
	}
	if err != nil {
		return err
	}
	if categoryID == nil {
		return nil
	}
	category, err := s.categoryRepo.FindByID(ctx, *categoryID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && category.PCategoryID != parentID) {
		return &BusinessError{Info: "\u4e8c\u7ea7\u5206\u533a\u4e0d\u5b58\u5728\u6216\u4e0d\u5c5e\u4e8e\u5f53\u524d\u4e00\u7ea7\u5206\u533a"}
	}
	return err
}

func (s *VideoPostService) validatePartCount(ctx context.Context, count int) error {
	setting := domain.DefaultSysSetting()
	if s.settingStore != nil {
		stored, exists, err := s.settingStore.Get(ctx)
		if err != nil {
			return err
		}
		if exists {
			setting = stored
		}
	}
	setting = domain.NormalizeSysSetting(setting)
	if count > setting.VideoPCount {
		return &BusinessError{Info: fmt.Sprintf("\u89c6\u9891\u5206P\u6570\u91cf\u4e0d\u80fd\u8d85\u8fc7%d\u4e2a", setting.VideoPCount)}
	}
	return nil
}

func (s *VideoPostService) validateImageFiles(list []ImagePostUploadFile) error {
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		if _, duplicated := seen[item.SourceName]; duplicated {
			return &BusinessError{Info: "\u540c\u4e00\u5f20\u56fe\u7247\u4e0d\u80fd\u91cd\u590d\u63d0\u4ea4"}
		}
		seen[item.SourceName] = struct{}{}
		targetPath, err := s.safeImageResourcePath(item.SourceName)
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(targetPath)
		if err != nil || fileInfo.IsDir() {
			return &BusinessError{Info: "\u56fe\u7247\u6587\u4ef6\u4e0d\u5b58\u5728\uff0c\u8bf7\u91cd\u65b0\u4e0a\u4f20"}
		}
	}
	return nil
}

func (s *VideoPostService) safeImageResourcePath(sourceName string) (string, error) {
	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" || !safeRelativeResourceName(sourceName) {
		return "", &BusinessError{Info: "\u56fe\u7247\u5730\u5740\u4e0d\u6b63\u786e"}
	}
	if _, ok := imagePostAllowedExts[strings.ToLower(filepath.Ext(sourceName))]; !ok {
		return "", &BusinessError{Info: "\u4ec5\u652f\u6301 jpg\u3001jpeg\u3001png\u3001gif\u3001bmp\u3001webp \u683c\u5f0f\u56fe\u7247"}
	}
	return safeResourceSubdirectory(s.resourceRoot, sourceName)
}

func (s *VideoPostService) publishAfterCommit(messages []mq.VideoTranscodeMessage) {
	if s.publisher == nil {
		return
	}
	for _, message := range messages {
		message := message
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.publisher.PublishVideoTranscodeMessage(ctx, message); err == nil {
				_ = s.repository.MarkTranscodeMessagePublished(context.Background(), message.MessageID, time.Now().Add(-10*time.Minute))
			}
		}()
	}
}

func normalizeVideoPostInput(input SaveVideoPostInput) SaveVideoPostInput {
	input.UserID = strings.TrimSpace(input.UserID)
	input.VideoID = strings.TrimSpace(input.VideoID)
	input.VideoCover = strings.TrimSpace(input.VideoCover)
	input.VideoName = strings.TrimSpace(input.VideoName)
	input.OriginInfo = strings.TrimSpace(input.OriginInfo)
	input.Tags = strings.TrimSpace(input.Tags)
	input.Introduction = strings.TrimSpace(input.Introduction)
	input.Interaction = strings.TrimSpace(input.Interaction)
	for index := range input.UploadFileList {
		input.UploadFileList[index].UploadID = strings.TrimSpace(input.UploadFileList[index].UploadID)
		input.UploadFileList[index].FileID = strings.TrimSpace(input.UploadFileList[index].FileID)
		input.UploadFileList[index].FileName = strings.TrimSpace(input.UploadFileList[index].FileName)
	}
	for index := range input.ImageList {
		input.ImageList[index].FileID = strings.TrimSpace(input.ImageList[index].FileID)
		input.ImageList[index].SourceName = strings.TrimSpace(input.ImageList[index].SourceName)
		input.ImageList[index].FileName = strings.TrimSpace(input.ImageList[index].FileName)
	}
	if input.ContentType == domain.ContentTypeImage && input.VideoCover == "" && len(input.ImageList) > 0 {
		input.VideoCover = input.ImageList[0].SourceName
	}
	return input
}

func validateVideoPostInput(input SaveVideoPostInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "\u8bf7\u5148\u767b\u5f55"}
	}
	if input.VideoID != "" && (len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID)) {
		return &BusinessError{Info: "稿件ID不正确"}
	}
	if !isValidContentType(input.ContentType) {
		return &BusinessError{Info: "稿件类型不正确"}
	}
	if input.VideoCover == "" || utf8.RuneCountInString(input.VideoCover) > maxVideoCoverLength || !safeRelativeResourceName(input.VideoCover) {
		return &BusinessError{Info: "稿件封面不正确"}
	}
	if input.VideoName == "" || utf8.RuneCountInString(input.VideoName) > maxVideoNameLength {
		return &BusinessError{Info: "稿件标题不能为空且不能超过100个字符"}
	}
	if input.PCategoryID <= 0 || (input.PostType != 0 && input.PostType != 1) {
		return &BusinessError{Info: "稿件分区或投稿类型不正确"}
	}
	if input.PostType == 1 && input.OriginInfo == "" {
		return &BusinessError{Info: "转载稿件请填写来源"}
	}
	if utf8.RuneCountInString(input.OriginInfo) > maxVideoOriginInfoLength || input.Tags == "" || utf8.RuneCountInString(input.Tags) > maxVideoTagsLength || utf8.RuneCountInString(input.Introduction) > maxVideoIntroductionLength {
		return &BusinessError{Info: "稿件来源、标签或简介超出长度限制"}
	}
	if utf8.RuneCountInString(input.Interaction) > maxVideoInteractionLength || !validVideoInteraction(input.Interaction) {
		return &BusinessError{Info: "\u4e92\u52a8\u8bbe\u7f6e\u4e0d\u6b63\u786e"}
	}
	if input.DownloadPermission != 0 && input.DownloadPermission != 1 {
		return &BusinessError{Info: "下载设置不正确"}
	}
	if input.ContentType == domain.ContentTypeImage {
		return validateImagePostInput(input)
	}
	if len(input.UploadFileList) == 0 {
		return &BusinessError{Info: "\u8bf7\u81f3\u5c11\u4e0a\u4f20\u4e00\u4e2a\u89c6\u9891\u6587\u4ef6"}
	}
	for _, item := range input.UploadFileList {
		if item.FileName == "" || utf8.RuneCountInString(item.FileName) > maxVideoPartFileNameLength {
			return &BusinessError{Info: "\u89c6\u9891\u5206P\u540d\u79f0\u4e0d\u80fd\u4e3a\u7a7a\u4e14\u4e0d\u80fd\u8d85\u8fc7200\u4e2a\u5b57\u7b26"}
		}
		if item.FileID == "" && item.UploadID == "" {
			return &BusinessError{Info: "\u89c6\u9891\u6587\u4ef6\u53c2\u6570\u4e0d\u6b63\u786e"}
		}
		if item.FileID != "" && (len(item.FileID) != videoFileIDLength || !isAlphaNumeric(item.FileID)) {
			return &BusinessError{Info: "\u89c6\u9891\u5206P ID\u4e0d\u6b63\u786e"}
		}
	}
	return nil
}

func validateImagePostInput(input SaveVideoPostInput) error {
	if len(input.ImageList) == 0 {
		return &BusinessError{Info: "请至少上传一张图片"}
	}
	if len(input.ImageList) > maxImagePostCount {
		return &BusinessError{Info: "图片投稿最多支持9张图片"}
	}
	for _, item := range input.ImageList {
		if item.SourceName == "" || utf8.RuneCountInString(item.SourceName) > maxVideoCoverLength || !safeRelativeResourceName(item.SourceName) {
			return &BusinessError{Info: "图片地址不正确"}
		}
		if _, ok := imagePostAllowedExts[strings.ToLower(filepath.Ext(item.SourceName))]; !ok {
			return &BusinessError{Info: "仅支持 jpg、jpeg、png、gif、bmp、webp 格式图片"}
		}
		if item.FileName == "" || utf8.RuneCountInString(item.FileName) > maxVideoPartFileNameLength {
			return &BusinessError{Info: "图片名称不能为空且不能超过200个字符"}
		}
	}
	return nil
}

func buildVideoPost(input SaveVideoPostInput, videoID string, now time.Time, status int) domain.VideoInfoPost {
	post := domain.VideoInfoPost{
		VideoID: videoID, VideoCover: input.VideoCover, VideoName: input.VideoName, UserID: input.UserID,
		CreateTime: now, LastUpdateTime: now, PCategoryID: input.PCategoryID, CategoryID: input.CategoryID,
		ContentType: input.ContentType, Status: status, PostType: input.PostType, Tags: input.Tags, DownloadPermission: input.DownloadPermission,
	}
	if input.ContentType == domain.ContentTypeImage {
		duration := 0
		post.Duration = &duration
	}
	post.OriginInfo = optionalString(input.OriginInfo)
	post.Introduction = optionalString(input.Introduction)
	post.Interaction = optionalString(input.Interaction)
	return post
}

func videoPostMetadataChanged(current domain.VideoInfoPost, input SaveVideoPostInput) bool {
	return current.VideoCover != input.VideoCover || current.VideoName != input.VideoName ||
		current.PCategoryID != input.PCategoryID || !equalOptionalInt(current.CategoryID, input.CategoryID) ||
		current.ContentType != input.ContentType || current.PostType != input.PostType || optionalStringValue(current.OriginInfo) != input.OriginInfo ||
		current.Tags != input.Tags || optionalStringValue(current.Introduction) != input.Introduction ||
		optionalStringValue(current.Interaction) != input.Interaction ||
		current.DownloadPermission != input.DownloadPermission
}

func safeResourceSubdirectory(root string, relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative)))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", errors.New("invalid resource path")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", errors.New("resource path escapes root")
	}
	return targetAbs, nil
}

func safeRelativeResourceName(value string) bool {
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean != "." && !filepath.IsAbs(clean) && !strings.HasPrefix(clean, "..")
}

func validVideoInteraction(value string) bool {
	return value == "" || value == "0" || value == "1" || value == "0,1" || value == "1,0"
}

func isValidContentType(contentType int) bool {
	return contentType == domain.ContentTypeVideo || contentType == domain.ContentTypeImage
}

func isAlphaNumeric(value string) bool {
	for _, char := range value {
		if !strings.ContainsRune(uploadIDAlphabet, char) {
			return false
		}
	}
	return value != ""
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func equalOptionalInt(left *int, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
