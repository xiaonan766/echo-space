package web

import (
	"context"
	"errors"
	"strings"
)

type VideoOnlineReporter interface {
	Report(ctx context.Context, fileID string, deviceID string) (int, error)
}

type VideoOnlineService struct {
	store VideoOnlineReporter
}

type ReportVideoPlayOnlineInput struct {
	FileID   string
	DeviceID string
	ClientIP string
}

func NewVideoOnlineService(store VideoOnlineReporter) *VideoOnlineService {
	return &VideoOnlineService{store: store}
}

func (s *VideoOnlineService) ReportVideoPlayOnline(ctx context.Context, input ReportVideoPlayOnlineInput) (int, error) {
	input = normalizeReportVideoPlayOnlineInput(input)
	if err := validateReportVideoPlayOnlineInput(input); err != nil {
		return 0, err
	}
	if s == nil || s.store == nil {
		return 1, nil
	}
	return s.store.Report(ctx, input.FileID, input.DeviceID)
}

func normalizeReportVideoPlayOnlineInput(input ReportVideoPlayOnlineInput) ReportVideoPlayOnlineInput {
	input.FileID = strings.TrimSpace(input.FileID)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.ClientIP = strings.TrimSpace(input.ClientIP)
	if input.DeviceID == "" {
		if input.ClientIP == "" {
			input.DeviceID = "unknown"
		} else {
			input.DeviceID = "unknown:" + input.ClientIP
		}
	}
	return input
}

func validateReportVideoPlayOnlineInput(input ReportVideoPlayOnlineInput) error {
	if len(input.FileID) != videoFileIDLength || !isAlphaNumeric(input.FileID) {
		return &BusinessError{Info: "参数错误"}
	}
	if input.DeviceID == "" {
		return errors.New("video online device id is empty after normalization")
	}
	return nil
}
