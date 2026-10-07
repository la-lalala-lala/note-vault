package repository

import (
	"context"
	"errors"
	"sort"
	"strings"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/generated"
	"note-vault/app/model"
	"note-vault/app/utils"

	"gorm.io/cli/gorm/typed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db ...*gorm.DB) *RoleRepository {
	repository := &RoleRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *RoleRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return typed.G[model.Role](db).Create(ctx, role)
}

func (r *RoleRepository) FindByID(ctx context.Context, id uint) (*model.Role, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	role, err := typed.G[model.Role](db).Where(generated.Role.ID.Eq(id)).Take(ctx)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	db, err := r.database()
	if err != nil {
		return false, err
	}
	query := typed.G[model.Role](db).Where(generated.Role.Code.Eq(code))
	if excludeID > 0 {
		query = query.Where(generated.Role.ID.Neq(excludeID))
	}
	count, err := query.Count(ctx, "*")
	return count > 0, err
}

func (r *RoleRepository) List(ctx context.Context, input dto.RoleRequest) ([]model.Role, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}
	query := typed.G[model.Role](db).Scopes()
	if input.Status != nil {
		query = query.Where(generated.Role.Status.Eq(*input.Status))
	}
	total, err := query.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}
	roles, err := query.Order(generated.Role.ID.OrderExpr("? DESC", generated.Role.ID)).
		Offset((*input.Page - 1) * *input.PageSize).Limit(*input.PageSize).Find(ctx)
	return roles, total, err
}

func (r *RoleRepository) Update(ctx context.Context, id uint, input dto.RoleRequest) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments := make([]clause.Assigner, 0, 4)
	if input.Name != nil {
		assignments = append(assignments, generated.Role.Name.Set(strings.TrimSpace(*input.Name)))
	}
	if input.Code != nil {
		assignments = append(assignments, generated.Role.Code.Set(strings.TrimSpace(*input.Code)))
	}
	if input.Description != nil {
		assignments = append(assignments, generated.Role.Description.Set(strings.TrimSpace(*input.Description)))
	}
	if input.Status != nil {
		assignments = append(assignments, generated.Role.Status.Set(*input.Status))
	}
	if len(assignments) == 0 {
		return nil
	}
	_, err = typed.G[model.Role](db).Where(generated.Role.ID.Eq(id)).Set(assignments...).Update(ctx)
	return err
}

func (r *RoleRepository) Delete(ctx context.Context, id uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	_, err = typed.G[model.Role](db).Where(generated.Role.ID.Eq(id)).Delete(ctx)
	return err
}

func (r *RoleRepository) CountUsers(ctx context.Context, roleID uint) (int64, error) {
	db, err := r.database()
	if err != nil {
		return 0, err
	}
	return typed.G[model.UserRole](db).Where(generated.UserRole.RoleID.Eq(roleID)).Count(ctx, "*")
}

// FindMenusByRoleID 查询角色关联的全部菜单和权限资源，不过滤禁用状态，便于管理端完整回显授权关系。
func (r *RoleRepository) FindMenusByRoleID(ctx context.Context, roleID uint) ([]model.Menu, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	roleMenus, err := typed.G[model.RoleMenu](db).
		Where(generated.RoleMenu.RoleID.Eq(roleID)).Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(roleMenus) == 0 {
		return []model.Menu{}, nil
	}

	menuIDs := make([]uint, 0, len(roleMenus))
	menuIDSet := make(map[uint]struct{}, len(roleMenus))
	for _, roleMenu := range roleMenus {
		if _, exists := menuIDSet[roleMenu.MenuID]; exists {
			continue
		}
		menuIDSet[roleMenu.MenuID] = struct{}{}
		menuIDs = append(menuIDs, roleMenu.MenuID)
	}

	menus, err := typed.G[model.Menu](db).
		Where(generated.Menu.ID.In(menuIDs...)).
		Order(clause.OrderBy{Columns: []clause.OrderByColumn{
			{Column: generated.Menu.SortOrder.Column()},
			{Column: generated.Menu.ID.Column()},
		}}).Find(ctx)
	if err != nil {
		return nil, err
	}
	return menus, nil
}

func (r *RoleRepository) ReplaceMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除这个角色 的中间关系
		if _, err := typed.G[model.RoleMenu](tx).Where(generated.RoleMenu.RoleID.Eq(roleID)).Delete(ctx); err != nil {
			return err
		}
		if len(menuIDs) == 0 {
			return nil
		}
		items := make([]model.RoleMenu, 0, len(menuIDs))
		for _, menuID := range menuIDs {
			items = append(items, model.RoleMenu{RoleID: roleID, MenuID: menuID})
		}
		// 重新为角色 菜单批量绑定
		return typed.G[model.RoleMenu](tx).CreateInBatches(ctx, &items, len(items))
	})
}

func (r *RoleRepository) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除这个用户的角色
		if _, err := typed.G[model.UserRole](tx).Where(generated.UserRole.UserID.Eq(userID)).Delete(ctx); err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		items := make([]model.UserRole, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			items = append(items, model.UserRole{UserID: userID, RoleID: roleID})
		}
		// 重新为用户授权角色
		return typed.G[model.UserRole](tx).CreateInBatches(ctx, &items, len(items))
	})
}

func (r *RoleRepository) ExistsIDs(ctx context.Context, roleIDs []uint) (bool, error) {
	if len(roleIDs) == 0 {
		return true, nil
	}
	db, err := r.database()
	if err != nil {
		return false, err
	}
	count, err := typed.G[model.Role](db).Where(generated.Role.ID.In(roleIDs...)).Count(ctx, "*")
	return count == int64(len(roleIDs)), err
}

// FindByUserID 查询用户关联的全部角色，包括已禁用角色。
func (r *RoleRepository) FindByUserID(ctx context.Context, userID uint) ([]model.Role, error) {
	rolesByUserID, err := r.FindByUserIDs(ctx, []uint{userID})
	if err != nil {
		return nil, err
	}
	return rolesByUserID[userID], nil
}

// FindByUserIDs 批量查询多个用户关联的全部角色，避免用户列表产生逐用户查询。
func (r *RoleRepository) FindByUserIDs(ctx context.Context, userIDs []uint) (map[uint][]model.Role, error) {
	rolesByUserID := make(map[uint][]model.Role, len(userIDs))
	if len(userIDs) == 0 {
		return rolesByUserID, nil
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}

	userRoles, err := typed.G[model.UserRole](db).
		Where(generated.UserRole.UserID.In(userIDs...)).Find(ctx)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]uint, 0, len(userRoles))
	roleIDSet := make(map[uint]struct{}, len(userRoles))
	for _, userRole := range userRoles {
		if _, exists := roleIDSet[userRole.RoleID]; exists {
			continue
		}
		roleIDSet[userRole.RoleID] = struct{}{}
		roleIDs = append(roleIDs, userRole.RoleID)
	}
	if len(roleIDs) == 0 {
		return rolesByUserID, nil
	}

	roles, err := typed.G[model.Role](db).
		Where(generated.Role.ID.In(roleIDs...)).Find(ctx)
	if err != nil {
		return nil, err
	}
	roleByID := make(map[uint]model.Role, len(roles))
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	for _, userRole := range userRoles {
		role, exists := roleByID[userRole.RoleID]
		if !exists {
			continue
		}
		rolesByUserID[userRole.UserID] = append(rolesByUserID[userRole.UserID], role)
	}
	return rolesByUserID, nil
}

// AuthorizationDataByUserID 查询用户当前启用角色拥有的权限和菜单。
func (r *RoleRepository) AuthorizationDataByUserID(ctx context.Context, userID uint) ([]string, []model.Menu, bool, error) {
	db, err := r.database()
	if err != nil {
		return nil, nil, false, err
	}
	// 查询这个用户关联的角色
	userRoles, err := typed.G[model.UserRole](db).Where(generated.UserRole.UserID.Eq(userID)).Find(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	roleIDs := make([]uint, 0, len(userRoles))
	for _, userRole := range userRoles {
		roleIDs = append(roleIDs, userRole.RoleID)
	}
	// 获取角色id
	if len(roleIDs) == 0 {
		return []string{}, []model.Menu{}, false, nil
	}
	// 查询角色
	roles, err := typed.G[model.Role](db).Where(generated.Role.ID.In(roleIDs...)).Where(generated.Role.Status.Eq(model.RoleStatusEnabled)).Find(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	activeRoleIDs := make([]uint, 0, len(roles))
	superAdmin := false
	// 遍历角色，判断是不是超管，获取角色id
	for _, role := range roles {
		if role.Code == model.RoleCodeSuperAdmin {
			superAdmin = true
		}
		activeRoleIDs = append(activeRoleIDs, role.ID)
	}
	if len(activeRoleIDs) == 0 {
		if superAdmin {
			return []string{utils.AllAPIPermissions}, []model.Menu{}, true, nil
		}
		return []string{}, []model.Menu{}, false, nil
	}
	// 关联角色菜单中间表，查询菜单
	roleMenus, err := typed.G[model.RoleMenu](db).Where(generated.RoleMenu.RoleID.In(activeRoleIDs...)).Find(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	menuIDs := make([]uint, 0, len(roleMenus))
	for _, roleMenu := range roleMenus {
		menuIDs = append(menuIDs, roleMenu.MenuID)
	}
	// 获取菜单id列表
	if len(menuIDs) == 0 {
		if superAdmin {
			return []string{utils.AllAPIPermissions}, []model.Menu{}, true, nil
		}
		return []string{}, []model.Menu{}, false, nil
	}
	// 查询菜单
	menus, err := typed.G[model.Menu](db).
		Where(generated.Menu.ID.In(menuIDs...)).
		Where(generated.Menu.Type.In(model.MenuTypeDirectory, model.MenuTypePage, model.MenuTypeButton, model.MenuTypeAPI)).
		Where(generated.Menu.Status.Eq(model.MenuStatusEnabled)).
		Order(clause.OrderBy{Columns: []clause.OrderByColumn{
			{Column: generated.Menu.SortOrder.Column()},
			{Column: generated.Menu.ID.Column()},
		}}).Find(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	permissionSet := make(map[string]struct{}, len(menus))
	permissions := make([]string, 0, len(menus))
	menuTreeItems := make([]model.Menu, 0, len(menus))
	for _, menu := range menus {
		if menu.Type != model.MenuTypeAPI {
			menuTreeItems = append(menuTreeItems, menu)
			continue
		}
		if menu.HTTPMethod == nil || menu.APIPath == nil {
			continue
		}
		permission := strings.ToUpper(strings.TrimSpace(*menu.HTTPMethod)) + ":" + strings.TrimSpace(*menu.APIPath)
		if _, exists := permissionSet[permission]; exists {
			continue
		}
		permissionSet[permission] = struct{}{}
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	if superAdmin {
		permissions = []string{utils.AllAPIPermissions}
	}
	return permissions, menuTreeItems, superAdmin, nil
}
