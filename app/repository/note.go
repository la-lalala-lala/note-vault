package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/generated"
	"note-vault/app/model"
	"note-vault/app/utils"

	"gorm.io/cli/gorm/typed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db ...*gorm.DB) *NoteRepository {
	repository := &NoteRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *NoteRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

func (r *NoteRepository) Create(ctx context.Context, note *model.Note) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.Note](db).Create(ctx, note)
}

// CountCreatedBetween 统计当前用户指定时间范围内创建的笔记数量。
func (r *NoteRepository) CountCreatedBetween(ctx context.Context, ownerID uint, start, end time.Time) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Note](db).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.CreatedAt.Gte(start)).
		Where(generated.Note.CreatedAt.Lt(end)).
		Count(ctx, "*")
}

// CountByOwner 统计当前用户拥有的笔记总数，包括回收站中的笔记。
func (r *NoteRepository) CountByOwner(ctx context.Context, ownerID uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Note](db).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Count(ctx, "*")
}

// FindByID 使用资源 ID 和当前用户 ID 查询正常笔记，避免读取其他用户或回收站中的数据。
func (r *NoteRepository) FindByID(ctx context.Context, ownerID, id uint) (*model.Note, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	note, err := typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// CountByNotebookIDs 批量统计当前用户指定笔记簿下的笔记数（正常状态中）。
func (r *NoteRepository) CountByNotebookIDs(ctx context.Context, ownerID uint, notebookIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(notebookIDs))
	if len(notebookIDs) == 0 {
		return counts, nil
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		NotebookID uint  `gorm:"column:notebook_id"`
		NoteCount  int64 `gorm:"column:note_count"`
	}
	query := typed.G[model.Note](db).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.NotebookID.In(notebookIDs...)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
		Select(
			generated.Note.NotebookID,
			generated.Note.ID.SelectExpr("COUNT(?) AS ?", generated.Note.ID.Column(), clause.Column{Name: "note_count"}),
		).
		Group(generated.Note.NotebookID)
	err = query.Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.NotebookID] = row.NoteCount
	}
	return counts, nil
}

// List 查询当前用户的正常笔记，只选择列表展示需要的摘要字段，不加载正文。
func (r *NoteRepository) List(ctx context.Context, ownerID uint, input dto.NoteRequest) ([]model.Note, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Note](db).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal))
	if input.NotebookID != nil {
		query = query.Where(generated.Note.NotebookID.Eq(*input.NotebookID))
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) != "" {
		pattern := "%" + strings.TrimSpace(*input.Title) + "%"
		query = query.Where(generated.Note.Title.Like(pattern))
	}
	if input.Label != nil && strings.TrimSpace(*input.Label) != "" {
		pattern := "%" + strings.TrimSpace(*input.Label) + "%"
		query = query.Where(generated.Note.Label.Expr(
			"COALESCE(?, '') LIKE ?",
			generated.Note.Label.Column(),
			pattern,
		))
	}
	if input.UpdatedFrom != nil {
		query = query.Where(generated.Note.UpdatedAt.Gte(*input.UpdatedFrom))
	}
	if input.UpdatedTo != nil {
		query = query.Where(generated.Note.UpdatedAt.Lte(*input.UpdatedTo))
	}

	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	notes, err := query.
		Select(
			generated.Note.ID,
			generated.Note.NotebookID,
			generated.Note.Title,
			generated.Note.Label,
			generated.Note.Summary,
			generated.Note.CoverURL,
			generated.Note.ContentType,
			generated.Note.Status,
			generated.Note.Version,
			generated.Note.CreatedAt,
			generated.Note.UpdatedAt,
		).
		Order(generated.Note.UpdatedAt.OrderExpr("? DESC", generated.Note.UpdatedAt)).
		Order(generated.Note.ID.OrderExpr("? DESC", generated.Note.ID)).
		Offset((*input.Page - 1) * *input.PageSize).
		Limit(*input.PageSize).
		Find(ctx)
	return notes, total, err
}

// ListTrash 只查询当前用户的回收站笔记，并且不加载正文内容。
func (r *NoteRepository) ListTrash(ctx context.Context, ownerID uint, page, pageSize int) ([]model.Note, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Note](db).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusTrash))
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	notes, err := query.
		Select(
			generated.Note.ID,
			generated.Note.NotebookID,
			generated.Note.Title,
			generated.Note.Label,
			generated.Note.Summary,
			generated.Note.CoverURL,
			generated.Note.ContentType,
			generated.Note.Status,
			generated.Note.Version,
			generated.Note.DeletedAt,
			generated.Note.PurgeAt,
			generated.Note.CreatedAt,
			generated.Note.UpdatedAt,
		).
		Order(generated.Note.DeletedAt.OrderExpr("? DESC", generated.Note.DeletedAt)).
		Order(generated.Note.ID.OrderExpr("? DESC", generated.Note.ID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(ctx)
	return notes, total, err
}

// FindTrashedByID 按笔记 ID 和用户 ID 查询笔记，并确保目标处于回收站。
func (r *NoteRepository) FindTrashedByID(ctx context.Context, ownerID, id uint) (*model.Note, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	note, err := typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	if note.Status != model.NoteStatusTrash {
		return nil, utils.ErrNoteNotTrashed
	}
	return &note, nil
}

// MoveToTrash 标记正常笔记为回收站状态，并保存删除人及清理时间。
func (r *NoteRepository) MoveToTrash(ctx context.Context, ownerID, id, deletedBy uint, deletedAt, purgeAt time.Time) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
		Set(
			generated.Note.Status.Set(model.NoteStatusTrash),
			generated.Note.DeletedAt.Set(deletedAt),
			generated.Note.DeletedBy.Set(deletedBy),
			generated.Note.PurgeAt.Set(purgeAt),
			generated.Note.Version.SetExpr(generated.Note.Version.Incr(1)),
			generated.Note.UpdatedAt.Set(deletedAt),
		).Update(ctx)
	if err != nil || rowsAffected > 0 {
		return err
	}

	note, err := typed.G[model.Note](db).
		Select(generated.Note.Status).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Take(ctx)
	if err != nil {
		return err
	}
	if note.Status == model.NoteStatusTrash {
		return utils.ErrNoteAlreadyTrashed
	}
	return gorm.ErrRecordNotFound
}

// Restore 从回收站恢复笔记并清空删除信息；外层 Service 已校验目标笔记簿。
func (r *NoteRepository) Restore(ctx context.Context, ownerID, id, notebookID uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusTrash)).
		Set(
			generated.Note.NotebookID.Set(notebookID),
			generated.Note.Status.Set(model.NoteStatusNormal),
			generated.Note.DeletedAt.SetExpr(clause.Expr{SQL: "NULL"}),
			generated.Note.DeletedBy.SetExpr(clause.Expr{SQL: "NULL"}),
			generated.Note.PurgeAt.SetExpr(clause.Expr{SQL: "NULL"}),
			generated.Note.Version.SetExpr(generated.Note.Version.Incr(1)),
			generated.Note.UpdatedAt.Set(time.Now().UTC()),
		).Update(ctx)
	if err != nil || rowsAffected > 0 {
		return err
	}

	_, err = typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Take(ctx)
	if err != nil {
		return err
	}
	return utils.ErrNoteNotTrashed
}

// PermanentDelete 仅物理删除回收站笔记；目标已不存在时按幂等成功处理。
func (r *NoteRepository) PermanentDelete(ctx context.Context, ownerID, id uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		note, err := typed.G[model.Note](tx).
			Scopes(func(statement *gorm.Statement) {
				statement.AddClause(clause.Locking{Strength: "UPDATE"})
			}).
			Where(generated.Note.ID.Eq(id)).
			Where(generated.Note.OwnerID.Eq(ownerID)).
			Take(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if note.Status != model.NoteStatusTrash {
			return utils.ErrNoteNotTrashed
		}
		_, err = typed.G[model.Note](tx).
			Where(generated.Note.ID.Eq(id)).
			Where(generated.Note.OwnerID.Eq(ownerID)).
			Where(generated.Note.Status.Eq(model.NoteStatusTrash)).
			Delete(ctx)
		return err
	})
}

// CleanupExpiredTrash 在一个事务中锁定一批到期回收站笔记并逐条物理删除。
// 每条删除使用独立保存点，单条失败时回滚该条操作并继续处理同一批次的其他笔记。
func (r *NoteRepository) CleanupExpiredTrash(ctx context.Context, now time.Time, batchSize int) (int, int, int, error) {
	db, err := r.database()
	if err != nil {
		return 0, 0, 0, err
	}

	scanned, succeeded, failed := 0, 0, 0
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		notes, err := typed.G[model.Note](tx).
			Scopes(func(statement *gorm.Statement) {
				statement.AddClause(clause.Locking{Strength: "UPDATE"})
			}).
			Select(generated.Note.ID).
			Where(generated.Note.Status.Eq(model.NoteStatusTrash)).
			Where(generated.Note.PurgeAt.Lte(now)).
			Order(generated.Note.PurgeAt.OrderExpr("? ASC", generated.Note.PurgeAt)).
			Order(generated.Note.ID.OrderExpr("? ASC", generated.Note.ID)).
			Limit(batchSize).
			Find(ctx)
		if err != nil {
			return err
		}
		scanned = len(notes)

		for index, note := range notes {
			savepoint := fmt.Sprintf("trash_cleanup_%d", index)
			if err := tx.SavePoint(savepoint).Error; err != nil {
				return err
			}

			rowsAffected, err := typed.G[model.Note](tx).
				Where(generated.Note.ID.Eq(note.ID)).
				Where(generated.Note.Status.Eq(model.NoteStatusTrash)).
				Where(generated.Note.PurgeAt.Lte(now)).
				Delete(ctx)
			if err == nil && rowsAffected == 1 {
				succeeded++
				continue
			}

			failed++
			if rollbackErr := tx.RollbackTo(savepoint).Error; rollbackErr != nil {
				if err != nil {
					return errors.Join(err, rollbackErr)
				}
				return rollbackErr
			}
		}
		return nil
	})
	if err != nil {
		return scanned, succeeded, failed, err
	}
	return scanned, succeeded, failed, nil
}

// Update 使用资源 ID、当前用户 ID 和版本号更新正常笔记，并在成功后将版本号加一。
func (r *NoteRepository) Update(ctx context.Context, ownerID, id, version uint, input dto.NoteRequest) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments := make([]clause.Assigner, 0, 9)
	if input.NotebookID != nil {
		assignments = append(assignments, generated.Note.NotebookID.Set(*input.NotebookID))
	}
	if input.Title != nil {
		assignments = append(assignments, generated.Note.Title.Set(strings.TrimSpace(*input.Title)))
	}
	if input.Label != nil {
		assignments = append(assignments, generated.Note.Label.Set(strings.TrimSpace(*input.Label)))
	}
	if input.Summary != nil {
		assignments = append(assignments, generated.Note.Summary.Set(strings.TrimSpace(*input.Summary)))
	}
	if input.CoverURL != nil {
		assignments = append(assignments, generated.Note.CoverURL.Set(strings.TrimSpace(*input.CoverURL)))
	}
	if input.Content != nil {
		assignments = append(assignments, generated.Note.Content.Set(*input.Content))
	}
	if input.ContentType != nil {
		assignments = append(assignments, generated.Note.ContentType.Set(strings.ToLower(strings.TrimSpace(*input.ContentType))))
	}
	assignments = append(
		assignments,
		generated.Note.Version.SetExpr(generated.Note.Version.Incr(1)),
		generated.Note.UpdatedAt.Set(time.Now().UTC()),
	)

	rowsAffected, err := typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
		Where(generated.Note.Version.Eq(version)).
		Set(assignments...).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		return nil
	}

	_, err = typed.G[model.Note](db).
		Where(generated.Note.ID.Eq(id)).
		Where(generated.Note.OwnerID.Eq(ownerID)).
		Where(generated.Note.Status.Eq(model.NoteStatusNormal)).
		Take(ctx)
	if err != nil {
		return err
	}
	return utils.ErrOptimisticLockConflict
}
