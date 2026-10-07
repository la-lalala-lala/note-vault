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

type MenuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db ...*gorm.DB) *MenuRepository {
	repository := &MenuRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *MenuRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

func (r *MenuRepository) Create(ctx context.Context, menu *model.Menu) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.Menu](db).Create(ctx, menu)
}

func (r *MenuRepository) FindByID(ctx context.Context, id uint) (*model.Menu, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	menu, err := typed.G[model.Menu](db).Where(generated.Menu.ID.Eq(id)).Take(ctx)
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (r *MenuRepository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	db, err := r.database()
	if err != nil {
		return false, err
	}
	query := typed.G[model.Menu](db).Where(generated.Menu.Code.Eq(code))
	if excludeID > 0 {
		query = query.Where(generated.Menu.ID.Neq(excludeID))
	}
	count, err := query.Count(ctx, "*")
	return count > 0, err
}

func (r *MenuRepository) List(ctx context.Context, input dto.MenuRequest) ([]model.Menu, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	query := typed.G[model.Menu](db).Scopes()
	if input.Status != nil {
		query = query.Where(generated.Menu.Status.Eq(*input.Status))
	}
	if input.Type != nil {
		query = query.Where(generated.Menu.Type.Eq(*input.Type))
	}
	return query.Order(clause.OrderBy{Columns: []clause.OrderByColumn{
		{Column: generated.Menu.SortOrder.Column()},
		{Column: generated.Menu.ID.Column()},
	}}).Find(ctx)
}

func (r *MenuRepository) Update(ctx context.Context, id uint, input dto.MenuRequest) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments := make([]clause.Assigner, 0, 10)
	if input.Name != nil {
		assignments = append(assignments, generated.Menu.Name.Set(strings.TrimSpace(*input.Name)))
	}
	if input.Code != nil {
		assignments = append(assignments, generated.Menu.Code.Set(strings.TrimSpace(*input.Code)))
	}
	if input.Type != nil {
		assignments = append(assignments, generated.Menu.Type.Set(*input.Type))
	}
	if input.RoutePath != nil {
		assignments = append(assignments, generated.Menu.RoutePath.Set(strings.TrimSpace(*input.RoutePath)))
	}
	if input.Component != nil {
		assignments = append(assignments, generated.Menu.Component.Set(strings.TrimSpace(*input.Component)))
	}
	if input.HTTPMethod != nil {
		assignments = append(assignments, generated.Menu.HTTPMethod.Set(strings.ToUpper(strings.TrimSpace(*input.HTTPMethod))))
	}
	if input.APIPath != nil {
		assignments = append(assignments, generated.Menu.APIPath.Set(strings.TrimSpace(*input.APIPath)))
	}
	if input.SortOrder != nil {
		assignments = append(assignments, generated.Menu.SortOrder.Set(*input.SortOrder))
	}
	if input.Visible != nil {
		assignments = append(assignments, generated.Menu.Visible.Set(*input.Visible))
	}
	if input.Status != nil {
		assignments = append(assignments, generated.Menu.Status.Set(*input.Status))
	}
	query := typed.G[model.Menu](db).Where(generated.Menu.ID.Eq(id))
	if input.ParentID != nil {
		if *input.ParentID == 0 {
			assignments = append(assignments, clause.Assignment{Column: generated.Menu.ParentID.Column(), Value: nil})
		} else {
			assignments = append(assignments, generated.Menu.ParentID.Set(*input.ParentID))
		}
	}
	if len(assignments) == 0 {
		return nil
	}
	_, err = query.Set(assignments...).Update(ctx)
	return err
}

func (r *MenuRepository) Delete(ctx context.Context, id uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	_, err = typed.G[model.Menu](db).Where(generated.Menu.ID.Eq(id)).Delete(ctx)
	return err
}

func (r *MenuRepository) CountChildren(ctx context.Context, id uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.Menu](db).Where(generated.Menu.ParentID.Eq(id)).Count(ctx, "*")
}

func (r *MenuRepository) ExistsIDs(ctx context.Context, menuIDs []uint) (bool, error) {
	if len(menuIDs) == 0 {
		return true, nil
	}
	db, err := r.database()
	if err != nil {
		return false, err
	}
	count, err := typed.G[model.Menu](db).Where(generated.Menu.ID.In(menuIDs...)).Count(ctx, "*")
	return count == int64(len(menuIDs)), err
}
