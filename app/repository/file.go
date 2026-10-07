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
)

// FileRepository 负责文件元数据的数据库读写，不处理文件对象内容。
type FileRepository struct {
	db *gorm.DB
}

// NewFileRepository 创建文件元数据仓储；未传入数据库时使用应用共享连接。
func NewFileRepository(db ...*gorm.DB) *FileRepository {
	repository := &FileRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *FileRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

// Create 创建文件元数据记录。
func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.File](db).Create(ctx, file)
}

// CountCreatedBetween 统计当前用户指定时间范围内上传的文件数量。
func (r *FileRepository) CountCreatedBetween(ctx context.Context, ownerID uint, start, end time.Time) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.File](db).
		Where(generated.File.OwnerID.Eq(ownerID)).
		Where(generated.File.CreatedAt.Gte(start)).
		Where(generated.File.CreatedAt.Lt(end)).
		Count(ctx, "*")
}

// CountByOwner 统计当前用户拥有的文件总数，包括已删除的文件记录。
func (r *FileRepository) CountByOwner(ctx context.Context, ownerID uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.File](db).
		Where(generated.File.OwnerID.Eq(ownerID)).
		Count(ctx, "*")
}

// FindByID 查询当前用户的正常文件，所有权条件始终与 ID 条件同时生效。
func (r *FileRepository) FindByID(ctx context.Context, ownerID, id uint) (*model.File, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	file, err := typed.G[model.File](db).
		Where(generated.File.ID.Eq(id)).
		Where(generated.File.OwnerID.Eq(ownerID)).
		Where(generated.File.Status.Eq(model.FileStatusNormal)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// FindPublicByKey 查询可公开预览的正常文件，不校验文件所属用户。
func (r *FileRepository) FindPublicByKey(ctx context.Context, publicKey string) (*model.File, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	file, err := typed.G[model.File](db).
		Where(generated.File.PublicKey.Eq(publicKey)).
		Where(generated.File.Status.Eq(model.FileStatusNormal)).
		Take(ctx)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// List 分页查询当前用户的正常文件，并支持文件名和 MIME 类型筛选。
func (r *FileRepository) List(ctx context.Context, ownerID uint, input dto.FileRequest) ([]model.File, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.File](db).
		Where(generated.File.OwnerID.Eq(ownerID)).
		Where(generated.File.Status.Eq(model.FileStatusNormal))
	if input.FileName != nil && strings.TrimSpace(*input.FileName) != "" {
		pattern := "%" + strings.TrimSpace(*input.FileName) + "%"
		query = query.Where(generated.File.FileName.Like(pattern))
	}
	if input.FileType != nil && strings.TrimSpace(*input.FileType) != "" {
		query = query.Where(generated.File.FileType.Eq(strings.TrimSpace(*input.FileType)))
	}

	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	files, err := query.
		Select(
			generated.File.ID,
			generated.File.FileName,
			generated.File.PublicKey,
			generated.File.FileURL,
			generated.File.FileType,
			generated.File.FileSize,
			generated.File.Checksum,
			generated.File.CreatedAt,
			generated.File.UpdatedAt,
		).
		Order(generated.File.CreatedAt.OrderExpr("? DESC", generated.File.CreatedAt)).
		Order(generated.File.ID.OrderExpr("? DESC", generated.File.ID)).
		Offset((*input.Page - 1) * *input.PageSize).
		Limit(*input.PageSize).
		Find(ctx)
	return files, total, err
}

// SoftDelete 将当前用户的正常文件标记为删除，并保留元数据记录。
func (r *FileRepository) SoftDelete(ctx context.Context, ownerID, id uint, deletedAt time.Time) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	rowsAffected, err := typed.G[model.File](db).
		Where(generated.File.ID.Eq(id)).
		Where(generated.File.OwnerID.Eq(ownerID)).
		Where(generated.File.Status.Eq(model.FileStatusNormal)).
		Set(
			generated.File.Status.Set(model.FileStatusDeleted),
			generated.File.DeletedAt.Set(deletedAt),
			generated.File.UpdatedAt.Set(deletedAt),
		).
		Update(ctx)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
