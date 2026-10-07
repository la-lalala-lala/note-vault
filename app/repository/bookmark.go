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

// BookmarkRepository 负责收藏夹数据的数据库读写，不处理请求参数和业务规则。
type BookmarkRepository struct {
	db *gorm.DB
}

// NewBookmarkRepository 创建收藏夹仓储；未传入数据库时使用应用共享连接。
func NewBookmarkRepository(db ...*gorm.DB) *BookmarkRepository {
	repository := &BookmarkRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *BookmarkRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

// Create 创建收藏夹记录。
func (r *BookmarkRepository) Create(ctx context.Context, bookmark *model.Bookmark) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.Bookmark](db).Create(ctx, bookmark)
}

// CountCreatedBetween 统计当前用户指定时间范围内新增的收藏数量。
func (r *BookmarkRepository) CountCreatedBetween(ctx context.Context, ownerID uint, start, end time.Time) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Bookmark](db).
		Where(generated.Bookmark.OwnerID.Eq(ownerID)).
		Where(generated.Bookmark.CreatedAt.Gte(start)).
		Where(generated.Bookmark.CreatedAt.Lt(end)).
		Count(ctx, "*")
}

// CountByOwner 统计当前用户拥有的收藏总数。
func (r *BookmarkRepository) CountByOwner(ctx context.Context, ownerID uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Bookmark](db).
		Where(generated.Bookmark.OwnerID.Eq(ownerID)).
		Count(ctx, "*")
}

// FindByID 查询当前用户自己的收藏夹记录。
func (r *BookmarkRepository) FindByID(ctx context.Context, ownerID, id uint) (*model.Bookmark, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	bookmark, err := typed.G[model.Bookmark](db).
		Where(generated.Bookmark.ID.Eq(id)).
		Where(generated.Bookmark.OwnerID.Eq(ownerID)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &bookmark, nil
}

// List 分页查询当前用户的收藏夹，并支持标题、标签、链接、重点状态和时间范围筛选。
func (r *BookmarkRepository) List(ctx context.Context, ownerID uint, input dto.BookmarkRequest) ([]model.Bookmark, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Bookmark](db).
		Where(generated.Bookmark.OwnerID.Eq(ownerID))
	if input.Topic != nil && strings.TrimSpace(*input.Topic) != "" {
		pattern := "%" + strings.TrimSpace(*input.Topic) + "%"
		query = query.Where(generated.Bookmark.Topic.Like(pattern))
	}
	if input.Label != nil && strings.TrimSpace(*input.Label) != "" {
		pattern := "%" + strings.TrimSpace(*input.Label) + "%"
		query = query.Where(generated.Bookmark.Label.Like(pattern))
	}
	if input.Favorite != nil {
		query = query.Where(generated.Bookmark.Favorite.Eq(*input.Favorite))
	}
	if input.CreatedFrom != nil {
		query = query.Where(generated.Bookmark.CreatedAt.Gte(*input.CreatedFrom))
	}
	if input.CreatedTo != nil {
		query = query.Where(generated.Bookmark.CreatedAt.Lte(*input.CreatedTo))
	}
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	bookmarks, err := query.
		Order(generated.Bookmark.UpdatedAt.OrderExpr("? DESC", generated.Bookmark.UpdatedAt)).
		Order(generated.Bookmark.ID.OrderExpr("? DESC", generated.Bookmark.ID)).
		Offset((*input.Page - 1) * *input.PageSize).
		Limit(*input.PageSize).
		Find(ctx)
	return bookmarks, total, err
}

// Update 按当前用户和收藏夹 ID 部分更新收藏夹字段。
func (r *BookmarkRepository) Update(ctx context.Context, ownerID, id uint, input dto.BookmarkRequest) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments := make([]clause.Assigner, 0, 7)
	if input.Topic != nil {
		assignments = append(assignments, generated.Bookmark.Topic.Set(strings.TrimSpace(*input.Topic)))
	}
	if input.Label != nil {
		assignments = append(assignments, generated.Bookmark.Label.Set(strings.TrimSpace(*input.Label)))
	}
	if input.Href != nil {
		assignments = append(assignments, generated.Bookmark.Href.Set(strings.TrimSpace(*input.Href)))
	}
	if input.Icon != nil {
		assignments = append(assignments, generated.Bookmark.Icon.Set(strings.TrimSpace(*input.Icon)))
	}
	if input.Favorite != nil {
		assignments = append(assignments, generated.Bookmark.Favorite.Set(*input.Favorite))
	}
	if input.Description != nil {
		assignments = append(assignments, generated.Bookmark.Description.Set(strings.TrimSpace(*input.Description)))
	}
	if len(assignments) == 0 {
		return nil
	}
	assignments = append(assignments, generated.Bookmark.UpdatedAt.Set(time.Now().UTC()))
	rowsAffected, err := typed.G[model.Bookmark](db).
		Where(generated.Bookmark.ID.Eq(id)).
		Where(generated.Bookmark.OwnerID.Eq(ownerID)).
		Set(assignments...).Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete 删除当前用户自己的收藏夹记录。
func (r *BookmarkRepository) Delete(ctx context.Context, ownerID, id uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.Bookmark](db).
		Where(generated.Bookmark.ID.Eq(id)).
		Where(generated.Bookmark.OwnerID.Eq(ownerID)).
		Delete(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
