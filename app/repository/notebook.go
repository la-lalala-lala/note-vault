package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/generated"
	"note-vault/app/model"

	"gorm.io/cli/gorm/typed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotebookRepository struct {
	db *gorm.DB
}

func NewNotebookRepository(db ...*gorm.DB) *NotebookRepository {
	repository := &NotebookRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *NotebookRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

func (r *NotebookRepository) Create(ctx context.Context, notebook *model.Notebook) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.Notebook](db).Create(ctx, notebook)
}

// CountCreatedBetween 统计当前用户指定时间范围内创建的笔记簿数量。
func (r *NotebookRepository) CountCreatedBetween(ctx context.Context, ownerID uint, start, end time.Time) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Notebook](db).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.CreatedAt.Gte(start)).
		Where(generated.Notebook.CreatedAt.Lt(end)).
		Count(ctx, "*")
}

// CountByOwner 统计当前用户拥有的笔记簿总数，包括已删除的笔记簿记录。
func (r *NotebookRepository) CountByOwner(ctx context.Context, ownerID uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Notebook](db).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Count(ctx, "*")
}

// FindByID 使用资源 ID 和当前用户 ID 查询未删除笔记簿，避免读取其他用户的数据。
func (r *NotebookRepository) FindByID(ctx context.Context, ownerID, id uint) (*model.Notebook, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	notebook, err := typed.G[model.Notebook](db).
		Where(generated.Notebook.ID.Eq(id)).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull()).Take(ctx)
	if err != nil {
		return nil, err
	}
	return &notebook, nil
}

// ExistsByName 判断同一用户下是否存在同名的未删除笔记簿，比较时忽略大小写。
func (r *NotebookRepository) ExistsByName(ctx context.Context, ownerID uint, name string, excludeID uint) (bool, error) {
	db, err := r.database()
	if err != nil {
		return false, err
	}
	query := typed.G[model.Notebook](db).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull()).
		Where(generated.Notebook.Name.Expr(
			"LOWER(?) = LOWER(?)",
			generated.Notebook.Name.Column(),
			strings.TrimSpace(name),
		))
	if excludeID > 0 {
		query = query.Where(generated.Notebook.ID.Neq(excludeID))
	}
	count, err := query.Count(ctx, "*")
	return count > 0, err
}

// List 查询当前用户的未删除笔记簿，所有筛选条件都建立在 owner_id 约束之上。
func (r *NotebookRepository) List(ctx context.Context, ownerID uint, input dto.NotebookRequest) ([]model.Notebook, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Notebook](db).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull())
	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		pattern := "%" + strings.TrimSpace(*input.Name) + "%"
		query = query.Where(generated.Notebook.Name.Expr(
			"LOWER(?) LIKE LOWER(?)",
			generated.Notebook.Name.Column(),
			pattern,
		))
	}
	if input.Tag != nil && strings.TrimSpace(*input.Tag) != "" {
		pattern := "%" + strings.TrimSpace(*input.Tag) + "%"
		query = query.Where(generated.Notebook.Tag.Expr(
			"LOWER(?) LIKE LOWER(?)",
			generated.Notebook.Tag.Column(),
			pattern,
		))
	}
	if input.Status != nil {
		query = query.Where(generated.Notebook.Status.Eq(*input.Status))
	}
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	notebooks, err := query.
		Order(generated.Notebook.SortOrder.OrderExpr("? ASC", generated.Notebook.SortOrder)).
		Order(generated.Notebook.ID.OrderExpr("? ASC", generated.Notebook.ID)).
		Offset((*input.Page - 1) * *input.PageSize).
		Limit(*input.PageSize).
		Find(ctx)
	return notebooks, total, err
}

// Update 使用资源 ID 和当前用户 ID 修改未删除笔记簿的基本资料。
func (r *NotebookRepository) Update(ctx context.Context, ownerID, id uint, input dto.NotebookRequest) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments := make([]clause.Assigner, 0, 4)
	if input.Name != nil {
		assignments = append(assignments, generated.Notebook.Name.Set(strings.TrimSpace(*input.Name)))
	}
	if input.Icon != nil {
		assignments = append(assignments, generated.Notebook.Icon.Set(*input.Icon))
	}
	if input.Tag != nil {
		assignments = append(assignments, generated.Notebook.Tag.Set(strings.TrimSpace(*input.Tag)))
	}
	if input.Description != nil {
		assignments = append(assignments, generated.Notebook.Description.Set(strings.TrimSpace(*input.Description)))
	}
	if len(assignments) == 0 {
		return nil
	}
	rowsAffected, err := typed.G[model.Notebook](db).
		Where(generated.Notebook.ID.Eq(id)).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull()).
		Set(assignments...).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateSort 使用资源 ID 和当前用户 ID 修改未删除笔记簿的排序值。
func (r *NotebookRepository) UpdateSort(ctx context.Context, ownerID, id uint, sortOrder int) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Notebook](db).
		Where(generated.Notebook.ID.Eq(id)).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull()).
		Set(generated.Notebook.SortOrder.Set(sortOrder)).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateStatus 使用资源 ID 和当前用户 ID修改未删除笔记簿的状态。
func (r *NotebookRepository) UpdateStatus(ctx context.Context, ownerID, id uint, status int16) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Notebook](db).
		Where(generated.Notebook.ID.Eq(id)).
		Where(generated.Notebook.OwnerID.Eq(ownerID)).
		Where(generated.Notebook.DeletedAt.IsNull()).
		Set(generated.Notebook.Status.Set(status)).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteIfEmpty 在事务中锁定当前用户的笔记簿；只有笔记簿中不存在任何笔记记录时才将其标记为删除。
// 正常笔记和回收站笔记都是仍然存在的记录，因此无需按状态分别统计。
func (r *NotebookRepository) DeleteIfEmpty(ctx context.Context, ownerID, id uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	var noteCount int64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 在删除笔记簿前，查询并锁定目标笔记簿记录
		_, err := typed.G[model.Notebook](tx).
			Scopes(func(statement *gorm.Statement) {
				statement.AddClause(clause.Locking{Strength: "UPDATE"})
			}).
			Where(generated.Notebook.ID.Eq(id)).
			Where(generated.Notebook.OwnerID.Eq(ownerID)).
			Where(generated.Notebook.DeletedAt.IsNull()).Take(ctx)
		if err != nil {
			return err
		}

		noteCount, err = typed.G[model.Note](tx).
			Where(generated.Note.OwnerID.Eq(ownerID)).
			Where(generated.Note.NotebookID.Eq(id)).
			Count(ctx, "*")
		if err != nil {
			return err
		}
		if noteCount > 0 {
			return nil
		}

		now := time.Now().UTC()
		rowsAffected, err := typed.G[model.Notebook](tx).
			Where(generated.Notebook.ID.Eq(id)).
			Where(generated.Notebook.OwnerID.Eq(ownerID)).
			Where(generated.Notebook.DeletedAt.IsNull()).
			Set(
				generated.Notebook.Status.Set(model.NotebookStatusDeleted),
				generated.Notebook.DeletedAt.Set(now),
			).Update(ctx)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	return noteCount, err
}
