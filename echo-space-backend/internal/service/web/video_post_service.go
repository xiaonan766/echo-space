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
	videoIDLength              = 10
	videoFileIDLength          = 20
	videoMessageIDLength       = 32
	maxVideoCoverLength        = 255
	maxVideoNameLength         = 100
	maxVideoTagsLength         = 300
	maxVideoIntroductionLength = 2000
	maxVideoOriginInfoLength   = 200
	maxVideoInteractionLength  = 3
	maxVideoPartFileNameLength = 200
)

type VideoTranscodePublisher interface {
	PublishVideoTranscodeMessage(ctx context.Context, message mq.VideoTranscodeMessage) error
}

type VideoPostUploadFile struct {
	UploadID  string `json:"uploadId"`
	FileID    string `json:"fileId"`
	FileName  string `json:"fileName"`
	FileIndex int    `json:"-"`
}

type SaveVideoPostInput struct {
	UserID         string
	VideoID        string
	VideoCover     string
	VideoName      string
	PCategoryID    int
	CategoryID     *int
	PostType       int
	OriginInfo     string
	Tags           string
	Introduction   string
	Interaction    string
	UploadFileList []VideoPostUploadFile
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
	if s == nil || s.repository == nil || s.categoryRepo == nil || s.uploadStore == nil {
		return "", errors.New("video post service is not ready")
	}
	if err := s.validateCategory(ctx, input.PCategoryID, input.CategoryID); err != nil {
		return "", err
	}
	if err := s.validatePartCount(ctx, len(input.UploadFileList)); err != nil {
		return "", err
	}

	if input.VideoID == "" {
		return s.createVideoPost(ctx, input)
	}
	return s.updateVideoPost(ctx, input)
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
	if setting.VideoPCount <= 0 {
		setting.VideoPCount = domain.DefaultSysSetting().VideoPCount
	}
	if count > setting.VideoPCount {
		return &BusinessError{Info: fmt.Sprintf("\u89c6\u9891\u5206P\u6570\u91cf\u4e0d\u80fd\u8d85\u8fc7%d\u4e2a", setting.VideoPCount)}
	}
	return nil
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
	return input
}

func validateVideoPostInput(input SaveVideoPostInput) error {
	if input.UserID == "" {
		return &BusinessError{Info: "\u8bf7\u5148\u767b\u5f55"}
	}
	if input.VideoID != "" && (len(input.VideoID) != videoIDLength || !isAlphaNumeric(input.VideoID)) {
		return &BusinessError{Info: "\u89c6\u9891ID\u4e0d\u6b63\u786e"}
	}
	if input.VideoCover == "" || utf8.RuneCountInString(input.VideoCover) > maxVideoCoverLength || !safeRelativeResourceName(input.VideoCover) {
		return &BusinessError{Info: "\u89c6\u9891\u5c01\u9762\u4e0d\u6b63\u786e"}
	}
	if input.VideoName == "" || utf8.RuneCountInString(input.VideoName) > maxVideoNameLength {
		return &BusinessError{Info: "\u89c6\u9891\u6807\u9898\u4e0d\u80fd\u4e3a\u7a7a\u4e14\u4e0d\u80fd\u8d85\u8fc7100\u4e2a\u5b57\u7b26"}
	}
	if input.PCategoryID <= 0 || (input.PostType != 0 && input.PostType != 1) {
		return &BusinessError{Info: "\u89c6\u9891\u5206\u533a\u6216\u6295\u7a3f\u7c7b\u578b\u4e0d\u6b63\u786e"}
	}
	if input.PostType == 1 && input.OriginInfo == "" {
		return &BusinessError{Info: "\u8f6c\u8f7d\u89c6\u9891\u8bf7\u586b\u5199\u6765\u6e90"}
	}
	if utf8.RuneCountInString(input.OriginInfo) > maxVideoOriginInfoLength || input.Tags == "" || utf8.RuneCountInString(input.Tags) > maxVideoTagsLength || utf8.RuneCountInString(input.Introduction) > maxVideoIntroductionLength {
		return &BusinessError{Info: "\u89c6\u9891\u6765\u6e90\u3001\u6807\u7b7e\u6216\u7b80\u4ecb\u8d85\u51fa\u957f\u5ea6\u9650\u5236"}
	}
	if utf8.RuneCountInString(input.Interaction) > maxVideoInteractionLength || !validVideoInteraction(input.Interaction) {
		return &BusinessError{Info: "\u4e92\u52a8\u8bbe\u7f6e\u4e0d\u6b63\u786e"}
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

func buildVideoPost(input SaveVideoPostInput, videoID string, now time.Time, status int) domain.VideoInfoPost {
	post := domain.VideoInfoPost{
		VideoID: videoID, VideoCover: input.VideoCover, VideoName: input.VideoName, UserID: input.UserID,
		CreateTime: now, LastUpdateTime: now, PCategoryID: input.PCategoryID, CategoryID: input.CategoryID,
		Status: status, PostType: input.PostType, Tags: input.Tags,
	}
	post.OriginInfo = optionalString(input.OriginInfo)
	post.Introduction = optionalString(input.Introduction)
	post.Interaction = optionalString(input.Interaction)
	return post
}

func videoPostMetadataChanged(current domain.VideoInfoPost, input SaveVideoPostInput) bool {
	return current.VideoCover != input.VideoCover || current.VideoName != input.VideoName ||
		current.PCategoryID != input.PCategoryID || !equalOptionalInt(current.CategoryID, input.CategoryID) ||
		current.PostType != input.PostType || optionalStringValue(current.OriginInfo) != input.OriginInfo ||
		current.Tags != input.Tags || optionalStringValue(current.Introduction) != input.Introduction ||
		optionalStringValue(current.Interaction) != input.Interaction
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
