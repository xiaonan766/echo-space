package web

import (
	"context"
	"testing"
)

func TestReportVideoPlayOnlineRejectsInvalidFileID(t *testing.T) {
	service := NewVideoOnlineService(&fakeVideoOnlineStore{})

	_, err := service.ReportVideoPlayOnline(context.Background(), ReportVideoPlayOnlineInput{
		FileID:   "bad",
		DeviceID: "device-1",
	})
	if err == nil {
		t.Fatal("expected invalid file id error")
	}
	if _, ok := IsBusinessError(err); !ok {
		t.Fatalf("error = %#v, want business error", err)
	}
}

func TestReportVideoPlayOnlineFallbacksEmptyDeviceID(t *testing.T) {
	store := &fakeVideoOnlineStore{count: 3}
	service := NewVideoOnlineService(store)

	count, err := service.ReportVideoPlayOnline(context.Background(), ReportVideoPlayOnlineInput{
		FileID:   "VM1M5t5IsNy9bLRWM1ji",
		ClientIP: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("report video play online returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
	if store.deviceID != "unknown:127.0.0.1" {
		t.Fatalf("deviceID = %q, want fallback with client IP", store.deviceID)
	}
}

func TestReportVideoPlayOnlineReturnsStoreCount(t *testing.T) {
	store := &fakeVideoOnlineStore{count: 7}
	service := NewVideoOnlineService(store)

	count, err := service.ReportVideoPlayOnline(context.Background(), ReportVideoPlayOnlineInput{
		FileID:   "VM1M5t5IsNy9bLRWM1ji",
		DeviceID: "device-1",
	})
	if err != nil {
		t.Fatalf("report video play online returned error: %v", err)
	}
	if count != 7 {
		t.Fatalf("count = %d, want 7", count)
	}
	if store.fileID != "VM1M5t5IsNy9bLRWM1ji" || store.deviceID != "device-1" {
		t.Fatalf("store input = (%q, %q), want original file/device", store.fileID, store.deviceID)
	}
}

type fakeVideoOnlineStore struct {
	fileID   string
	deviceID string
	count    int
	err      error
}

func (s *fakeVideoOnlineStore) Report(ctx context.Context, fileID string, deviceID string) (int, error) {
	s.fileID = fileID
	s.deviceID = deviceID
	if s.err != nil {
		return 0, s.err
	}
	return s.count, nil
}
