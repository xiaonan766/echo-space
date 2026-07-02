package repository

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}
	return db
}

func TestApplyApprovedGalleryImageFilter(t *testing.T) {
	db := newDryRunDB(t)
	stmt := applyApprovedGalleryImageFilter(db.Table("video_info_post v")).
		Find(&[]domain.GalleryImageItem{}).Statement

	sql := stmt.SQL.String()
	if !strings.Contains(sql, "COALESCE(v.content_type, 0) = ?") {
		t.Fatalf("sql missing content type filter: %s", sql)
	}
	if !strings.Contains(sql, "v.status = ?") {
		t.Fatalf("sql missing approved status filter: %s", sql)
	}
	wantVars := []any{domain.ContentTypeImage, domain.VideoPostStatusApproved}
	if !reflect.DeepEqual(stmt.Vars, wantVars) {
		t.Fatalf("vars = %#v, want %#v", stmt.Vars, wantVars)
	}
}

func TestApplyApprovedGalleryImageFileFilter(t *testing.T) {
	db := newDryRunDB(t)
	stmt := applyApprovedGalleryImageFileFilter(db.Table("video_info_file_post"), "Abc123Def4").
		Find(&[]domain.GalleryImageFile{}).Statement

	sql := stmt.SQL.String()
	for _, pattern := range []string{
		"video_id = ?",
		"update_type <> ?",
		"transfer_result = ?",
		"COALESCE(file_path, '') <> ''",
	} {
		if !strings.Contains(sql, pattern) {
			t.Fatalf("sql missing %q filter: %s", pattern, sql)
		}
	}
	wantVars := []any{"Abc123Def4", domain.VideoFileUpdateDeletePending, domain.VideoFileTransferSuccess}
	if !reflect.DeepEqual(stmt.Vars, wantVars) {
		t.Fatalf("vars = %#v, want %#v", stmt.Vars, wantVars)
	}
}
