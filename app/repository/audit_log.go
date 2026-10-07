package repository

import (
	"context"
	"errors"
	"strings"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/generated"
	"note-vault/app/model"

	"gorm.io/cli/gorm/typed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db ...*gorm.DB) *AuditLogRepository {
	result := &AuditLogRepository{}
	if len(db) > 0 {
		result.db = db[0]
	}
	return result
}

func (r *AuditLogRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

// Create 写入不可修改的审计记录，operated_at 由数据库默认值生成。
func (r *AuditLogRepository) Create(ctx context.Context, entry *model.AuditLog) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.AuditLog](db).Create(ctx, entry)
}

func (r *AuditLogRepository) List(ctx context.Context, input dto.AuditLogRequest) ([]model.AuditLog, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.AuditLog](db).Scopes()
	if input.OperatorAccount != nil && strings.TrimSpace(*input.OperatorAccount) != "" {
		query = query.Where(generated.AuditLog.OperatorAccount.Eq(strings.TrimSpace(*input.OperatorAccount)))
	}
	if input.IP != nil && *input.IP != "" {
		query = query.Where(generated.AuditLog.IP.Eq(*input.IP))
	}
	if input.Operation != nil && strings.TrimSpace(*input.Operation) != "" {
		query = query.Where(generated.AuditLog.Operation.Like("%" + strings.TrimSpace(*input.Operation) + "%"))
	}
	if input.OperatedFrom != nil {
		query = query.Where(generated.AuditLog.OperatedAt.Gte(*input.OperatedFrom))
	}
	if input.OperatedTo != nil {
		query = query.Where(generated.AuditLog.OperatedAt.Lte(*input.OperatedTo))
	}
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	logs, err := query.Order(clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: generated.AuditLog.OperatedAt.Column(), Desc: true},
		{Column: generated.AuditLog.ID.Column(), Desc: true},
	}}).
		Offset((*input.Page - 1) * (*input.PageSize)).Limit(*input.PageSize).Find(ctx)
	return logs, total, err
}
