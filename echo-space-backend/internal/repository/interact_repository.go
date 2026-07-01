package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/domain"
)

var (
	ErrUserActionVideoNotFound    = errors.New("user action video not found")
	ErrUserActionCommentNotFound  = errors.New("user action comment not found")
	ErrUserActionSelfCoin         = errors.New("user action self coin")
	ErrUserActionCoinUsed         = errors.New("user action coin used")
	ErrUserActionInsufficientCoin = errors.New("user action insufficient coin")
	ErrUserActionCoinFailed       = errors.New("user action coin failed")
)

type InteractRepository struct {
	db *gorm.DB
}

type InteractListQuery struct {
	PageNo         int
	PageSize       int
	VideoNameFuzzy string
}

type CommentCursorQuery struct {
	VideoID       string
	PCommentID    int
	OrderType     int
	Limit         int
	LastCommentID int
	LastLikeCount int
}

type UcenterInteractListQuery struct {
	UserID    string
	VideoID   string
	CursorID  int
	Direction string
	Limit     int
}

func NewInteractRepository(db *gorm.DB) *InteractRepository {
	return &InteractRepository{
		db: db,
	}
}

func (r *InteractRepository) ListCommentByPage(ctx context.Context, query InteractListQuery) ([]domain.AdminCommentItem, int64, error) {
	var totalCount int64
	countDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_comment vc"), "vc", query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var comments []domain.AdminCommentItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_comment vc"), "vc", query)
	err := listDB.
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			COALESCE(vi.video_cover, '') AS video_cover,
			vc.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(vc.reply_user_id, '') AS reply_user_id,
			COALESCE(reply_user.nick_name, '') AS reply_nick_name,
			COALESCE(vc.content, '') AS content,
			COALESCE(vc.img_path, '') AS img_path,
			COALESCE(DATE_FORMAT(vc.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Joins("LEFT JOIN user_info reply_user ON vc.reply_user_id = reply_user.user_id").
		Order("vc.comment_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&comments).Error
	if err != nil {
		return nil, 0, err
	}
	return comments, totalCount, nil
}

func (r *InteractRepository) ListDanmuByPage(ctx context.Context, query InteractListQuery) ([]domain.AdminDanmuItem, int64, error) {
	var totalCount int64
	countDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_danmu vd"), "vd", query)
	if err := countDB.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	var danmuList []domain.AdminDanmuItem
	offset := (query.PageNo - 1) * query.PageSize
	listDB := r.applyVideoNameFilter(r.db.WithContext(ctx).Table("video_danmu vd"), "vd", query)
	err := listDB.
		Select(`
			vd.danmu_id,
			vd.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			vd.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			vd.time,
			COALESCE(vd.text, '') AS text,
			COALESCE(DATE_FORMAT(vd.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN user_info ui ON vd.user_id = ui.user_id").
		Order("vd.danmu_id desc").
		Offset(offset).
		Limit(query.PageSize).
		Scan(&danmuList).Error
	if err != nil {
		return nil, 0, err
	}
	return danmuList, totalCount, nil
}

func (r *InteractRepository) ListUcenterCommentByCursor(ctx context.Context, query UcenterInteractListQuery) ([]domain.UcenterCommentItem, error) {
	var comments []domain.UcenterCommentItem
	listDB := r.applyUcenterCommentFilter(r.db.WithContext(ctx).Table("video_comment vc"), query)
	if query.CursorID > 0 {
		if query.Direction == "prev" {
			listDB = listDB.Where("vc.comment_id > ?", query.CursorID)
		} else {
			listDB = listDB.Where("vc.comment_id < ?", query.CursorID)
		}
	}
	if query.Direction == "prev" {
		listDB = listDB.Order("vc.comment_id asc")
	} else {
		listDB = listDB.Order("vc.comment_id desc")
	}
	err := listDB.
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			COALESCE(vi.video_cover, '') AS video_cover,
			vc.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(vc.reply_user_id, '') AS reply_user_id,
			COALESCE(reply_user.nick_name, '') AS reply_nick_name,
			COALESCE(vc.content, '') AS content,
			COALESCE(vc.img_path, '') AS img_path,
			COALESCE(DATE_FORMAT(vc.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN video_info vi ON vc.video_id = vi.video_id").
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Joins("LEFT JOIN user_info reply_user ON vc.reply_user_id = reply_user.user_id").
		Limit(query.Limit).
		Scan(&comments).Error
	if err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []domain.UcenterCommentItem{}
	}
	return comments, nil
}

func (r *InteractRepository) ListUcenterDanmuByCursor(ctx context.Context, query UcenterInteractListQuery) ([]domain.UcenterDanmuItem, error) {
	var danmuList []domain.UcenterDanmuItem
	listDB := r.applyUcenterDanmuFilter(r.db.WithContext(ctx).Table("video_danmu vd"), query)
	if query.CursorID > 0 {
		if query.Direction == "prev" {
			listDB = listDB.Where("vd.danmu_id > ?", query.CursorID)
		} else {
			listDB = listDB.Where("vd.danmu_id < ?", query.CursorID)
		}
	}
	if query.Direction == "prev" {
		listDB = listDB.Order("vd.danmu_id asc")
	} else {
		listDB = listDB.Order("vd.danmu_id desc")
	}
	err := listDB.
		Select(`
			vd.danmu_id,
			vd.video_id,
			COALESCE(vi.video_name, '') AS video_name,
			COALESCE(vi.video_cover, '') AS video_cover,
			vd.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			vd.time,
			COALESCE(vd.text, '') AS text,
			COALESCE(DATE_FORMAT(vd.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time
		`).
		Joins("LEFT JOIN user_info ui ON vd.user_id = ui.user_id").
		Limit(query.Limit).
		Scan(&danmuList).Error
	if err != nil {
		return nil, err
	}
	if danmuList == nil {
		danmuList = []domain.UcenterDanmuItem{}
	}
	return danmuList, nil
}

func (r *InteractRepository) FindCommentDeleteInfo(ctx context.Context, commentID int) (*domain.CommentDeleteInfo, error) {
	var comment domain.CommentDeleteInfo
	err := r.db.WithContext(ctx).
		Table("video_comment").
		Select("comment_id, p_comment_id, video_id").
		Where("comment_id = ?", commentID).
		Take(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *InteractRepository) FindDanmuDeleteInfo(ctx context.Context, danmuID int) (*domain.DanmuDeleteInfo, error) {
	var danmu domain.DanmuDeleteInfo
	err := r.db.WithContext(ctx).
		Table("video_danmu").
		Select("danmu_id, video_id").
		Where("danmu_id = ?", danmuID).
		Take(&danmu).Error
	if err != nil {
		return nil, err
	}
	return &danmu, nil
}

func (r *InteractRepository) VideoExists(ctx context.Context, videoID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_info").
		Where("video_id = ?", videoID).
		Count(&count).Error
	return count > 0, err
}

func (r *InteractRepository) ListDanmu(ctx context.Context, videoID string, fileID string) ([]domain.WebDanmuItem, error) {
	var danmuList []domain.WebDanmuItem
	err := r.db.WithContext(ctx).
		Table("video_danmu").
		Select(`
			danmu_id,
			video_id,
			file_id,
			user_id,
			COALESCE(DATE_FORMAT(post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time,
			COALESCE(text, '') AS text,
			mode,
			COALESCE(color, '') AS color,
			time
		`).
		Where("video_id = ? AND file_id = ?", videoID, fileID).
		Order("danmu_id asc").
		Scan(&danmuList).Error
	return danmuList, err
}

func (r *InteractRepository) FindDanmuTarget(ctx context.Context, videoID string, fileID string) (*domain.DanmuTargetInfo, error) {
	var target domain.DanmuTargetInfo
	err := r.db.WithContext(ctx).
		Table("video_info vi").
		Select("vi.video_id, vf.file_id, COALESCE(vi.interaction, '') AS interaction").
		Joins("INNER JOIN video_info_file vf ON vf.video_id = vi.video_id AND vf.file_id = ?", fileID).
		Where("vi.video_id = ?", videoID).
		Take(&target).Error
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func (r *InteractRepository) FindCommentTarget(ctx context.Context, videoID string) (*domain.CommentTargetInfo, error) {
	var target domain.CommentTargetInfo
	err := r.db.WithContext(ctx).
		Table("video_info").
		Select("video_id, user_id AS video_user_id, COALESCE(interaction, '') AS interaction").
		Where("video_id = ?", videoID).
		Take(&target).Error
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func (r *InteractRepository) ListTopComments(ctx context.Context, videoID string) ([]domain.WebCommentItem, error) {
	var comments []domain.WebCommentItem
	err := r.webCommentBase(ctx).
		Where("vc.video_id = ? AND vc.p_comment_id = ? AND COALESCE(vc.top_type, 0) = ?", videoID, 0, 1).
		Order("vc.comment_id desc").
		Scan(&comments).Error
	return comments, err
}

func (r *InteractRepository) ListTopLevelCommentsByCursor(ctx context.Context, query CommentCursorQuery) ([]domain.WebCommentItem, error) {
	var comments []domain.WebCommentItem
	db := r.webCommentBase(ctx).
		Where("vc.video_id = ? AND vc.p_comment_id = ? AND COALESCE(vc.top_type, 0) <> ?", query.VideoID, 0, 1)

	if query.OrderType == 1 {
		if query.LastCommentID > 0 {
			db = db.Where("vc.comment_id < ?", query.LastCommentID)
		}
		db = db.Order("vc.comment_id desc")
	} else {
		if query.LastCommentID > 0 {
			db = db.Where("(vc.like_count < ? OR (vc.like_count = ? AND vc.comment_id < ?))", query.LastLikeCount, query.LastLikeCount, query.LastCommentID)
		}
		db = db.Order("vc.like_count desc, vc.comment_id desc")
	}

	err := db.Limit(query.Limit).Scan(&comments).Error
	return comments, err
}

func (r *InteractRepository) ListReplyCommentsByCursor(ctx context.Context, query CommentCursorQuery) ([]domain.WebCommentItem, error) {
	var comments []domain.WebCommentItem
	db := r.webCommentBase(ctx).
		Where("vc.video_id = ? AND vc.p_comment_id = ?", query.VideoID, query.PCommentID)
	if query.LastCommentID > 0 {
		db = db.Where("vc.comment_id > ?", query.LastCommentID)
	}

	err := db.Order("vc.comment_id asc").Limit(query.Limit).Scan(&comments).Error
	return comments, err
}

func (r *InteractRepository) CountTopLevelComments(ctx context.Context, videoID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_comment").
		Where("video_id = ? AND p_comment_id = ?", videoID, 0).
		Count(&count).Error
	return count, err
}

func (r *InteractRepository) CountReplyComments(ctx context.Context, videoID string, pCommentID int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("video_comment").
		Where("video_id = ? AND p_comment_id = ?", videoID, pCommentID).
		Count(&count).Error
	return count, err
}

func (r *InteractRepository) CountRepliesByParentIDs(ctx context.Context, parentIDs []int) (map[int]int, error) {
	countMap := make(map[int]int, len(parentIDs))
	if len(parentIDs) == 0 {
		return countMap, nil
	}

	var rows []struct {
		PCommentID int `gorm:"column:p_comment_id"`
		ReplyCount int `gorm:"column:reply_count"`
	}
	err := r.db.WithContext(ctx).
		Table("video_comment").
		Select("p_comment_id, COUNT(*) AS reply_count").
		Where("p_comment_id IN ?", parentIDs).
		Group("p_comment_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		countMap[row.PCommentID] = row.ReplyCount
	}
	return countMap, nil
}

func (r *InteractRepository) FindTopLevelComment(ctx context.Context, videoID string, commentID int) (*domain.CommentReplyInfo, error) {
	var comment domain.CommentReplyInfo
	err := r.db.WithContext(ctx).
		Table("video_comment vc").
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			vc.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar
		`).
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Where("vc.video_id = ? AND vc.comment_id = ? AND vc.p_comment_id = ?", videoID, commentID, 0).
		Take(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *InteractRepository) ListUserCommentActions(ctx context.Context, videoID string, userID string, commentIDs []int) ([]domain.UserActionItem, error) {
	if userID == "" || len(commentIDs) == 0 {
		return []domain.UserActionItem{}, nil
	}

	var actions []domain.UserActionItem
	err := r.db.WithContext(ctx).
		Table("user_action").
		Select(`
			action_id,
			video_id,
			comment_id,
			action_type,
			action_count,
			user_id,
			COALESCE(DATE_FORMAT(action_time, '%Y-%m-%d %H:%i:%s'), '') AS action_time
		`).
		Where("video_id = ? AND user_id = ? AND comment_id IN ? AND action_type IN ?", videoID, userID, commentIDs, []int{0, 1}).
		Order("action_id desc").
		Scan(&actions).Error
	if err != nil {
		return nil, err
	}
	if actions == nil {
		actions = []domain.UserActionItem{}
	}
	return actions, nil
}

func (r *InteractRepository) FindReplyComment(ctx context.Context, commentID int) (*domain.CommentReplyInfo, error) {
	var comment domain.CommentReplyInfo
	err := r.db.WithContext(ctx).
		Table("video_comment vc").
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			vc.user_id,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(ui.avatar, '') AS avatar
		`).
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Where("vc.comment_id = ?", commentID).
		Take(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *InteractRepository) CreateDanmu(ctx context.Context, danmu domain.VideoDanmu) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&danmu).Error; err != nil {
			return err
		}
		return tx.Table("video_info").
			Where("video_id = ?", danmu.VideoID).
			Update("danmu_count", gorm.Expr("danmu_count + ?", 1)).Error
	})
}

func (r *InteractRepository) CreateComment(ctx context.Context, comment *domain.VideoComment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		if comment.PCommentID != 0 {
			return nil
		}
		return tx.Table("video_info").
			Where("video_id = ?", comment.VideoID).
			Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	})
}

func (r *InteractRepository) SaveUserAction(ctx context.Context, action domain.UserAction) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var video domain.UserActionVideoTarget
		err := tx.Table("video_info").
			Select("video_id, user_id").
			Where("video_id = ?", action.VideoID).
			Take(&video).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserActionVideoNotFound
		}
		if err != nil {
			return err
		}

		action.VideoUserID = video.UserID
		if action.ActionTime.IsZero() {
			action.ActionTime = time.Now()
		}

		existingAction, err := findUserActionForUpdate(tx, action.VideoID, action.CommentID, action.UserID, action.ActionType)
		if err != nil {
			return err
		}

		switch action.ActionType {
		case 2, 3:
			return saveVideoToggleAction(tx, action, existingAction)
		case 4:
			return saveVideoCoinAction(tx, action, existingAction)
		case 0, 1:
			return saveCommentToggleAction(tx, action, existingAction)
		default:
			return nil
		}
	})
}

func (r *InteractRepository) DeleteComment(ctx context.Context, comment domain.CommentDeleteInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM video_comment WHERE comment_id = ?", comment.CommentID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if comment.PCommentID == 0 {
			if err := tx.Table("video_info").
				Where("video_id = ?", comment.VideoID).
				Update("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", 1)).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM video_comment WHERE p_comment_id = ?", comment.CommentID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *InteractRepository) DeleteDanmu(ctx context.Context, danmu domain.DanmuDeleteInfo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec("DELETE FROM video_danmu WHERE danmu_id = ?", danmu.DanmuID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.Table("video_info").
			Where("video_id = ?", danmu.VideoID).
			Update("danmu_count", gorm.Expr("GREATEST(danmu_count - ?, 0)", 1)).Error
	})
}

func (r *InteractRepository) applyVideoNameFilter(db *gorm.DB, tableAlias string, query InteractListQuery) *gorm.DB {
	db = db.Joins("LEFT JOIN video_info vi ON " + tableAlias + ".video_id = vi.video_id")
	if query.VideoNameFuzzy != "" {
		db = db.Where("vi.video_name LIKE ?", "%"+query.VideoNameFuzzy+"%")
	}
	return db
}

func (r *InteractRepository) applyUcenterCommentFilter(db *gorm.DB, query UcenterInteractListQuery) *gorm.DB {
	db = db.Where("vc.video_user_id = ?", query.UserID)
	if query.VideoID != "" {
		db = db.Where("vc.video_id = ?", query.VideoID)
	}
	return db
}

func (r *InteractRepository) applyUcenterDanmuFilter(db *gorm.DB, query UcenterInteractListQuery) *gorm.DB {
	db = db.Joins("INNER JOIN video_info vi ON vd.video_id = vi.video_id").
		Where("vi.user_id = ?", query.UserID)
	if query.VideoID != "" {
		db = db.Where("vd.video_id = ?", query.VideoID)
	}
	return db
}

func (r *InteractRepository) webCommentBase(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("video_comment vc").
		Select(`
			vc.comment_id,
			vc.p_comment_id,
			vc.video_id,
			vc.video_user_id,
			vc.user_id,
			COALESCE(ui.avatar, '') AS avatar,
			COALESCE(ui.nick_name, '') AS nick_name,
			COALESCE(vc.reply_user_id, '') AS reply_user_id,
			COALESCE(reply_user.avatar, '') AS reply_avatar,
			COALESCE(reply_user.nick_name, '') AS reply_nick_name,
			COALESCE(vc.content, '') AS content,
			COALESCE(vc.img_path, '') AS img_path,
			COALESCE(DATE_FORMAT(vc.post_time, '%Y-%m-%d %H:%i:%s'), '') AS post_time,
			COALESCE(vc.top_type, 0) AS top_type,
			COALESCE(vc.like_count, 0) AS like_count,
			COALESCE(vc.hate_count, 0) AS hate_count
		`).
		Joins("LEFT JOIN user_info ui ON vc.user_id = ui.user_id").
		Joins("LEFT JOIN user_info reply_user ON vc.reply_user_id = reply_user.user_id")
}

func findUserActionForUpdate(tx *gorm.DB, videoID string, commentID int, userID string, actionType int) (*domain.UserAction, error) {
	var action domain.UserAction
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("video_id = ? AND comment_id = ? AND user_id = ? AND action_type = ?", videoID, commentID, userID, actionType).
		Take(&action).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func saveVideoToggleAction(tx *gorm.DB, action domain.UserAction, existingAction *domain.UserAction) error {
	field, ok := videoActionCountField(action.ActionType)
	if !ok {
		return nil
	}

	changeCount := 1
	if existingAction != nil {
		if err := tx.Delete(existingAction).Error; err != nil {
			return err
		}
		changeCount = -1
	} else if err := tx.Create(&action).Error; err != nil {
		return err
	}

	return updateVideoCount(tx, action.VideoID, field, changeCount)
}

func saveVideoCoinAction(tx *gorm.DB, action domain.UserAction, existingAction *domain.UserAction) error {
	if action.VideoUserID == action.UserID {
		return ErrUserActionSelfCoin
	}
	if existingAction != nil {
		return ErrUserActionCoinUsed
	}

	result := tx.Table("user_info").
		Where("user_id = ? AND current_coin_count >= ?", action.UserID, action.ActionCount).
		Update("current_coin_count", gorm.Expr("current_coin_count - ?", action.ActionCount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserActionInsufficientCoin
	}

	result = tx.Table("user_info").
		Where("user_id = ?", action.VideoUserID).
		Updates(map[string]any{
			"current_coin_count": gorm.Expr("current_coin_count + ?", action.ActionCount),
			"total_coin_count":   gorm.Expr("total_coin_count + ?", action.ActionCount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserActionCoinFailed
	}

	if err := tx.Create(&action).Error; err != nil {
		return err
	}
	return updateVideoCount(tx, action.VideoID, "coin_count", action.ActionCount)
}

func saveCommentToggleAction(tx *gorm.DB, action domain.UserAction, existingAction *domain.UserAction) error {
	if err := ensureCommentBelongsToVideo(tx, action.VideoID, action.CommentID); err != nil {
		return err
	}

	opposeType := 1
	if action.ActionType == 1 {
		opposeType = 0
	}
	opposeAction, err := findUserActionForUpdate(tx, action.VideoID, action.CommentID, action.UserID, opposeType)
	if err != nil {
		return err
	}
	if opposeAction != nil {
		if err := tx.Delete(opposeAction).Error; err != nil {
			return err
		}
	}

	changeCount := 1
	if existingAction != nil {
		if err := tx.Delete(existingAction).Error; err != nil {
			return err
		}
		changeCount = -1
	} else if err := tx.Create(&action).Error; err != nil {
		return err
	}

	opposeChangeCount := 0
	if opposeAction != nil {
		opposeChangeCount = -1
	}

	actionField, _ := commentActionCountField(action.ActionType)
	opposeField, _ := commentActionCountField(opposeType)
	return updateCommentCount(tx, action.CommentID, actionField, changeCount, opposeField, opposeChangeCount)
}

func ensureCommentBelongsToVideo(tx *gorm.DB, videoID string, commentID int) error {
	var comment struct {
		CommentID int `gorm:"column:comment_id"`
	}
	err := tx.Table("video_comment").
		Select("comment_id").
		Where("video_id = ? AND comment_id = ?", videoID, commentID).
		Take(&comment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserActionCommentNotFound
	}
	return err
}

func updateVideoCount(tx *gorm.DB, videoID string, field string, delta int) error {
	result := tx.Table("video_info").
		Where("video_id = ?", videoID).
		Update(field, gorm.Expr("GREATEST(COALESCE("+field+", 0) + ?, 0)", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserActionVideoNotFound
	}
	return nil
}

func updateCommentCount(tx *gorm.DB, commentID int, actionField string, actionDelta int, opposeField string, opposeDelta int) error {
	result := tx.Table("video_comment").
		Where("comment_id = ?", commentID).
		Updates(map[string]any{
			actionField: gorm.Expr("GREATEST(COALESCE("+actionField+", 0) + ?, 0)", actionDelta),
			opposeField: gorm.Expr("GREATEST(COALESCE("+opposeField+", 0) + ?, 0)", opposeDelta),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserActionCommentNotFound
	}
	return nil
}

func videoActionCountField(actionType int) (string, bool) {
	switch actionType {
	case 2:
		return "like_count", true
	case 3:
		return "collect_count", true
	case 4:
		return "coin_count", true
	default:
		return "", false
	}
}

func commentActionCountField(actionType int) (string, bool) {
	switch actionType {
	case 0:
		return "like_count", true
	case 1:
		return "hate_count", true
	default:
		return "", false
	}
}
