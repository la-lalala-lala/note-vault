package service

import (
	"context"
	"fmt"
	"note-vault/app/utils"
	"sort"
	"strings"

	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
)

// RoleService 负责角色、菜单和授权关系的业务规则。
type RoleService struct {
	roles *repository.RoleRepository
	menus *repository.MenuRepository
	users *repository.UserRepository
}

func NewRoleService() *RoleService {
	return &RoleService{
		roles: repository.NewRoleRepository(),
		menus: repository.NewMenuRepository(),
		users: repository.NewUserRepository(),
	}
}

// 创建角色
func (s *RoleService) CreateRole(ctx context.Context, input dto.RoleRequest) (*dto.RoleResponse, error) {
	name, err := utils.RequiredText(input.Name, "角色名称")
	if err != nil {
		return nil, err
	}
	code, err := utils.RequiredText(input.Code, "角色编码")
	if err != nil {
		return nil, err
	}
	status := model.RoleStatusEnabled
	if input.Status != nil {
		status = *input.Status
	}
	if !utils.ValidRoleStatus(status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "角色状态无效")
	}
	// 重复性检查
	exists, err := s.roles.ExistsByCode(ctx, code, 0)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "检查角色编码失败")
	}
	if exists {
		return nil, utils.NewAppError(utils.KindConflict, "角色编码已存在")
	}
	role := model.Role{Name: name, Code: code, Description: input.Description, Status: status}
	if err := s.roles.Create(ctx, &role); err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "创建角色失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("创建角色 %s（ID %d）", role.Name, role.ID))
	result := dto.NewRoleResponse(role)
	return &result, nil
}

func (s *RoleService) ListRoles(ctx context.Context, input dto.RoleRequest) (*dto.PageResponse[dto.RoleResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	input.Page, input.PageSize = &page, &pageSize
	if input.Status != nil && !utils.ValidRoleStatus(*input.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "角色状态无效")
	}
	roles, total, err := s.roles.List(ctx, input)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取角色列表失败")
	}
	return &dto.PageResponse[dto.RoleResponse]{List: dto.NewRoleResponses(roles), Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *RoleService) GetRole(ctx context.Context, id uint) (*dto.RoleResponse, error) {
	role, err := s.roles.FindByID(ctx, id)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取角色详情失败")
	}
	menus, err := s.roles.FindMenusByRoleID(ctx, id)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取角色关联菜单失败")
	}
	result := dto.NewRoleResponseWithMenus(*role, menus)
	return &result, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, id uint, input dto.RoleRequest) (*dto.RoleResponse, error) {
	role, err := s.roles.FindByID(ctx, id)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取角色详情失败")
	}
	if input.Name == nil && input.Code == nil && input.Description == nil && input.Status == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少提供一个需要修改的字段")
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "角色名称不能为空")
	}
	if input.Code != nil {
		if role.Code == model.RoleCodeSuperAdmin || role.Code == model.RoleCodeUser {
			return nil, utils.NewAppError(utils.KindPermissionDenied, "内置角色编码不允许修改")
		}
		code := strings.TrimSpace(*input.Code)
		if code == "" {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "角色编码不能为空")
		}
		exists, err := s.roles.ExistsByCode(ctx, code, id)
		if err != nil {
			return nil, utils.BuildRoleRepositoryError(err, "检查角色编码失败")
		}
		if exists {
			return nil, utils.NewAppError(utils.KindConflict, "角色编码已存在")
		}
		*input.Code = code
	}
	if input.Status != nil && !utils.ValidRoleStatus(*input.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "角色状态无效")
	}
	if err := s.roles.Update(ctx, id, input); err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "更新角色失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改角色 %s（ID %d）的配置或状态", role.Name, id))
	return s.GetRole(ctx, id)
}

func (s *RoleService) UpdateRoleStatus(ctx context.Context, id uint, status int16) (*dto.RoleResponse, error) {
	return s.UpdateRole(ctx, id, dto.RoleRequest{Status: &status})
}

func (s *RoleService) DeleteRole(ctx context.Context, id uint) error {
	role, err := s.roles.FindByID(ctx, id)
	if err != nil {
		return utils.BuildRoleRepositoryError(err, "获取角色详情失败")
	}
	if role.Code == model.RoleCodeSuperAdmin || role.Code == model.RoleCodeUser {
		return utils.NewAppError(utils.KindPermissionDenied, "内置角色不允许删除")
	}
	count, err := s.roles.CountUsers(ctx, id)
	if err != nil {
		return utils.BuildRoleRepositoryError(err, "检查角色用户绑定失败")
	}
	if count > 0 {
		return utils.NewAppError(utils.KindConflict, "角色已绑定用户，不能删除")
	}
	if err := s.roles.Delete(ctx, id); err != nil {
		return utils.BuildRoleRepositoryError(err, "删除角色失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("删除角色 %s（ID %d）", role.Name, id))
	return nil
}

// 为用户授予角色
func (s *RoleService) AssignUserRoles(ctx context.Context, userID uint, input dto.IDsRequest) error {
	if _, err := s.users.FindByID(ctx, userID); err != nil {
		return utils.BuildRoleRepositoryError(err, "获取用户详情失败")
	}
	roleIDs := uniqueIDs(input.IDs)
	exists, err := s.roles.ExistsIDs(ctx, roleIDs)
	if err != nil {
		return utils.BuildRoleRepositoryError(err, "检查角色失败")
	}
	if !exists {
		return utils.NewAppError(utils.KindNotFound, "存在不存在的角色")
	}
	if err := s.roles.ReplaceUserRoles(ctx, userID, roleIDs); err != nil {
		return utils.BuildRoleRepositoryError(err, "更新用户角色失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("为用户 %d 分配 %d 个角色", userID, len(roleIDs)))
	return nil
}

// 为角色授予菜单
func (s *RoleService) AssignRoleMenus(ctx context.Context, roleID uint, input dto.IDsRequest) error {
	if _, err := s.roles.FindByID(ctx, roleID); err != nil {
		return utils.BuildRoleRepositoryError(err, "获取角色详情失败")
	}
	menuIDs := uniqueIDs(input.IDs)
	exists, err := s.menus.ExistsIDs(ctx, menuIDs)
	if err != nil {
		return utils.BuildRoleRepositoryError(err, "检查菜单权限失败")
	}
	if !exists {
		return utils.NewAppError(utils.KindNotFound, "存在不存在的菜单权限")
	}
	if err := s.roles.ReplaceMenus(ctx, roleID, menuIDs); err != nil {
		return utils.BuildRoleRepositoryError(err, "更新角色菜单权限失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("为角色 %d 分配 %d 项菜单权限", roleID, len(menuIDs)))
	return nil
}

// 创建菜单
func (s *RoleService) CreateMenu(ctx context.Context, input dto.MenuRequest) (*dto.MenuResponse, error) {
	name, err := utils.RequiredText(input.Name, "菜单名称")
	if err != nil {
		return nil, err
	}
	code, err := utils.RequiredText(input.Code, "菜单编码")
	if err != nil {
		return nil, err
	}
	if input.Type == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "菜单类型不能为空")
	}
	if err := s.validateMenu(ctx, 0, input); err != nil {
		return nil, err
	}
	exists, err := s.menus.ExistsByCode(ctx, code, 0)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "检查菜单编码失败")
	}
	if exists {
		return nil, utils.NewAppError(utils.KindConflict, "菜单编码已存在")
	}
	status := model.MenuStatusEnabled
	if input.Status != nil {
		status = *input.Status
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	visible := true
	if input.Visible != nil {
		visible = *input.Visible
	}
	parentID := input.ParentID
	if parentID != nil && *parentID == 0 {
		parentID = nil
	}
	var httpMethod *string
	if input.HTTPMethod != nil {
		normalizedHTTPMethod := strings.ToUpper(strings.TrimSpace(*input.HTTPMethod))
		httpMethod = &normalizedHTTPMethod
	}
	menu := model.Menu{ParentID: parentID, Name: name, Code: code, Type: *input.Type,
		RoutePath: input.RoutePath, Component: input.Component, HTTPMethod: httpMethod,
		APIPath: input.APIPath, SortOrder: sortOrder, Visible: visible, Status: status}
	if err := s.menus.Create(ctx, &menu); err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "创建菜单失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("创建菜单 %s", menu.Name))
	result := dto.NewMenuResponse(menu)
	return &result, nil
}

func (s *RoleService) ListMenuTree(ctx context.Context, input dto.MenuRequest) ([]dto.MenuResponse, error) {
	// 查询菜单列表
	menus, err := s.menus.List(ctx, input)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取菜单树失败")
	}
	// 转换成树
	return buildMenuTree(menus), nil
}

// 修改菜单
func (s *RoleService) UpdateMenu(ctx context.Context, id uint, input dto.MenuRequest) (*dto.MenuResponse, error) {
	// 找到原来的菜单
	current, err := s.menus.FindByID(ctx, id)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取菜单详情失败")
	}
	// 字段非空校验
	if input.Name == nil && input.Code == nil && input.ParentID == nil && input.Type == nil && input.RoutePath == nil &&
		input.Component == nil && input.HTTPMethod == nil && input.APIPath == nil && input.SortOrder == nil && input.Visible == nil && input.Status == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少提供一个需要修改的字段")
	}
	if input.Code != nil {
		code := strings.TrimSpace(*input.Code)
		if code == "" {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "菜单编码不能为空")
		}
		// 想更换code
		exists, err := s.menus.ExistsByCode(ctx, code, id)
		if err != nil {
			return nil, utils.BuildRoleRepositoryError(err, "检查菜单编码失败")
		}
		if exists {
			return nil, utils.NewAppError(utils.KindConflict, "菜单编码已存在")
		}
		*input.Code = code
	}
	if input.Type == nil {
		input.Type = &current.Type
	}
	if err := s.validateMenu(ctx, id, input); err != nil {
		return nil, err
	}
	if err := s.menus.Update(ctx, id, input); err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "更新菜单失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改菜单 %s（ID %d）", current.Name, id))
	// 返回新的菜单信息
	updated, err := s.menus.FindByID(ctx, id)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取菜单详情失败")
	}
	result := dto.NewMenuResponse(*updated)
	return &result, nil
}

func (s *RoleService) DeleteMenu(ctx context.Context, id uint) error {
	if _, err := s.menus.FindByID(ctx, id); err != nil {
		return utils.BuildRoleRepositoryError(err, "获取菜单详情失败")
	}
	count, err := s.menus.CountChildren(ctx, id)
	if err != nil {
		return utils.BuildRoleRepositoryError(err, "检查菜单子节点失败")
	}
	if count > 0 {
		return utils.NewAppError(utils.KindConflict, "菜单存在子节点，不能删除")
	}
	if err := s.menus.Delete(ctx, id); err != nil {
		return utils.BuildRoleRepositoryError(err, "删除菜单失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("删除菜单 ID %d", id))
	return nil
}

func (s *RoleService) validateMenu(ctx context.Context, menuID uint, input dto.MenuRequest) error {
	if input.Type == nil || *input.Type < model.MenuTypeDirectory || *input.Type > model.MenuTypeAPI {
		return utils.NewAppError(utils.KindInvalidArgument, "菜单类型无效")
	}
	if input.Status != nil && (*input.Status < model.MenuStatusEnabled || *input.Status > model.MenuStatusDisabled) {
		return utils.NewAppError(utils.KindInvalidArgument, "菜单状态无效")
	}
	// 校验新父菜单是否合法，避免菜单树出现循环引用
	// ParentID 非空且不为 0 时，表示要设置父菜单；nil 或 0 表示根节点
	if input.ParentID != nil && *input.ParentID != 0 {
		if *input.ParentID == menuID {
			return utils.NewAppError(utils.KindInvalidArgument, "菜单不能将自己设置为父节点")
		}
		if _, err := s.menus.FindByID(ctx, *input.ParentID); err != nil {
			return utils.BuildRoleRepositoryError(err, "菜单父节点不存在")
		}
		// map[uint]bool：表示一个 map，键的类型是 uint，值的类型是 bool， 创建 map 时放入一组键值，表示当前菜单的 ID 已访问
		visited := map[uint]bool{menuID: true}
		// 从指定父菜单开始，沿着它的 ParentID 一直向上查祖先
		parentID := *input.ParentID
		for {
			if parentID == 0 {
				break
			}
			if visited[parentID] {
				return utils.NewAppError(utils.KindInvalidArgument, "菜单父子关系存在循环引用")
			}
			visited[parentID] = true
			parent, err := s.menus.FindByID(ctx, parentID)
			if err != nil {
				return utils.BuildRoleRepositoryError(err, "菜单父节点不存在")
			}
			if parent.ParentID == nil {
				break
			}
			parentID = *parent.ParentID
		}
	}
	if *input.Type == model.MenuTypeAPI {
		httpMethod := input.HTTPMethod
		apiPath := input.APIPath
		if menuID > 0 {
			current, err := s.menus.FindByID(ctx, menuID)
			if err != nil {
				return utils.BuildRoleRepositoryError(err, "获取菜单详情失败")
			}
			if httpMethod == nil {
				httpMethod = current.HTTPMethod
			}
			if apiPath == nil {
				apiPath = current.APIPath
			}
		}
		if httpMethod == nil || strings.TrimSpace(*httpMethod) == "" || apiPath == nil || strings.TrimSpace(*apiPath) == "" {
			return utils.NewAppError(utils.KindInvalidArgument, "API 类型菜单必须配置 HTTP 方法和 API 路径")
		}
	}
	return nil
}

// 转换成树
func buildMenuTree(menus []model.Menu) []dto.MenuResponse {
	byParent := make(map[uint][]dto.MenuResponse)
	for _, menu := range menus {
		parentID := uint(0)
		if menu.ParentID != nil {
			parentID = *menu.ParentID
		}
		byParent[parentID] = append(byParent[parentID], dto.NewMenuResponse(menu))
	}
	var build func(uint) []dto.MenuResponse
	build = func(parentID uint) []dto.MenuResponse {
		items := byParent[parentID]
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SortOrder != items[j].SortOrder {
				return items[i].SortOrder < items[j].SortOrder
			}
			return items[i].ID < items[j].ID
		})
		for index := range items {
			items[index].Children = build(items[index].ID)
		}
		return items
	}
	return build(0)
}

// 去掉 0 和重复 ID，同时保留第一次出现的顺序
func uniqueIDs(ids []uint) []uint {
	result := make([]uint, 0, len(ids))
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
