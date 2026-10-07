package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"

	"golang.org/x/sync/errgroup"
)

const (
	defaultUserRoleCode = "USER"
)

// UserService 用户业务服务，负责用户业务规则、参数归一化和 DTO/Model 转换。
type UserService struct {
	users     *repository.UserRepository
	roles     *repository.RoleRepository
	notes     *repository.NoteRepository
	notebooks *repository.NotebookRepository
	files     *repository.FileRepository
	bookmarks *repository.BookmarkRepository
	redis     *config.RedisTemplate
}

// NewUserService 创建用户服务实例。
func NewUserService() *UserService {
	return &UserService{
		users:     repository.NewUserRepository(),
		roles:     repository.NewRoleRepository(),
		notes:     repository.NewNoteRepository(),
		notebooks: repository.NewNotebookRepository(),
		files:     repository.NewFileRepository(),
		bookmarks: repository.NewBookmarkRepository(),
		redis:     nil,
	}
}

// GetCurrentUserWeeklyStatistics 统计当前用户本周新增数量和各类资源总数。
func (s *UserService) GetCurrentUserWeeklyStatistics(ctx context.Context, ownerID uint) (*dto.UserWeeklyStatisticsResponse, error) {
	// 统一计算本周的时间范围，四个资源查询使用完全相同的起止时间。
	weekStart, weekEnd := currentWeekRange(time.Now())
	var (
		// 每个 goroutine 只写入自己对应的统计变量，避免并发写同一变量。
		weeklyNewNoteCount     int64
		totalNoteCount         int64
		weeklyNewNotebookCount int64
		totalNotebookCount     int64
		weeklyNewFileCount     int64
		totalFileCount         int64
		weeklyNewBookmarkCount int64
		totalBookmarkCount     int64
	)
	// 八项统计之间没有依赖关系，使用多个 goroutine 并行执行数据库查询。
	var group errgroup.Group
	group.Go(func() error {
		// 统计本周创建的笔记数量。
		count, err := s.notes.CountCreatedBetween(ctx, ownerID, weekStart, weekEnd)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计本周新增笔记失败", err)
		}
		weeklyNewNoteCount = count
		return nil
	})
	group.Go(func() error {
		// 统计当前用户拥有的笔记总数，包含回收站中的笔记。
		count, err := s.notes.CountByOwner(ctx, ownerID)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计笔记总数失败", err)
		}
		totalNoteCount = count
		return nil
	})
	group.Go(func() error {
		// 统计本周创建的笔记簿数量。
		count, err := s.notebooks.CountCreatedBetween(ctx, ownerID, weekStart, weekEnd)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计本周新增笔记簿失败", err)
		}
		weeklyNewNotebookCount = count
		return nil
	})
	group.Go(func() error {
		// 统计当前用户拥有的笔记簿总数，包含已删除的笔记簿记录。
		count, err := s.notebooks.CountByOwner(ctx, ownerID)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计笔记簿总数失败", err)
		}
		totalNotebookCount = count
		return nil
	})
	group.Go(func() error {
		// 统计本周上传的文件数量。
		count, err := s.files.CountCreatedBetween(ctx, ownerID, weekStart, weekEnd)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计本周新增文件失败", err)
		}
		weeklyNewFileCount = count
		return nil
	})
	group.Go(func() error {
		// 统计当前用户拥有的文件总数，包含已删除的文件记录。
		count, err := s.files.CountByOwner(ctx, ownerID)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计文件总数失败", err)
		}
		totalFileCount = count
		return nil
	})
	group.Go(func() error {
		// 统计本周新增的收藏数量。
		count, err := s.bookmarks.CountCreatedBetween(ctx, ownerID, weekStart, weekEnd)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计本周新增收藏失败", err)
		}
		weeklyNewBookmarkCount = count
		return nil
	})
	group.Go(func() error {
		// 统计当前用户拥有的收藏总数。
		count, err := s.bookmarks.CountByOwner(ctx, ownerID)
		if err != nil {
			return utils.WrapAppError(utils.KindInternal, "统计收藏总数失败", err)
		}
		totalBookmarkCount = count
		return nil
	})
	// 等待所有查询完成；任意一个查询失败都返回对应的业务错误。
	if err := group.Wait(); err != nil {
		return nil, err
	}
	// 所有查询成功后返回四类资源的本周新增数量和各自总数。
	return &dto.UserWeeklyStatisticsResponse{
		WeeklyNewNoteCount: weeklyNewNoteCount, TotalNoteCount: totalNoteCount,
		WeeklyNewNotebookCount: weeklyNewNotebookCount, TotalNotebookCount: totalNotebookCount,
		WeeklyNewFileCount: weeklyNewFileCount, TotalFileCount: totalFileCount,
		WeeklyNewBookmarkCount: weeklyNewBookmarkCount, TotalBookmarkCount: totalBookmarkCount,
	}, nil
}

func currentWeekRange(now time.Time) (time.Time, time.Time) {
	// 将时间转换为服务端本地时区，避免 UTC 日期与本地日期跨日造成统计偏差。
	localNow := now.In(time.Local)
	// Go 中 Sunday 的值为 0，通过取模计算当前日期距离周一的天数。
	daysSinceMonday := (int(localNow.Weekday()) + 6) % 7
	// 本周起点为周一 00:00，结束时间使用下周一 00:00，查询采用左闭右开区间。
	weekStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day()-daysSinceMonday, 0, 0, 0, 0, localNow.Location())
	return weekStart, weekStart.AddDate(0, 0, 7)
}

func (s *UserService) getRedisTemplate() *config.RedisTemplate {
	if nil == s.redis {
		s.redis = config.NewRedisTemplate(config.Redis, config.AppConfig.Redis.OperationTimeoutDuration())
	}
	return s.redis
}

// Register 注册用户，并将注册请求转换为普通用户创建请求。
func (s *UserService) Register(ctx context.Context, input dto.UserRequest) (*dto.UserResponse, error) {
	return s.createUser(ctx, input, false)
}

// CreateUser 创建管理员指定资料的用户。
func (s *UserService) CreateUser(ctx context.Context, input dto.UserRequest) (*dto.UserResponse, error) {
	return s.createUser(ctx, input, true)
}

func (s *UserService) createUser(ctx context.Context, input dto.UserRequest, allowAdminFields bool) (*dto.UserResponse, error) {
	if !allowAdminFields && (input.OpenID != nil || input.Sex != nil || input.Phone != nil || input.AvatarURL != nil || input.Status != nil) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "注册接口不允许设置管理员用户字段")
	}

	account, err := utils.RequiredText(input.Account, "账号")
	if err != nil {
		return nil, err
	}
	name, err := utils.RequiredText(input.Name, "姓名")
	if err != nil {
		return nil, err
	}
	password, err := utils.RequiredText(input.Password, "密码")
	if err != nil {
		return nil, err
	}

	status := model.UserStatusNormal
	if input.Status != nil {
		status = *input.Status
	}
	if !utils.ValidUserStatus(status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "用户状态无效")
	}

	exists, err := s.users.ExistsByAccount(ctx, account, 0)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "检查用户账号失败", err)
	}
	if exists {
		return nil, utils.NewAppError(utils.KindConflict, "用户账号已存在")
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInvalidArgument, "用户密码无效", err)
	}

	user := model.User{
		Account: account, Name: name, Password: passwordHash,
		OpenID: input.OpenID, Sex: input.Sex, Email: input.Email, Phone: input.Phone,
		AvatarURL: input.AvatarURL, Status: status, LoginMethod: model.LoginMethodPassword,
	}
	if err := s.users.CreateWithRole(ctx, &user, defaultUserRoleCode); err != nil {
		return nil, utils.BuildUserRepositoryError(err, "创建用户失败")
	}
	if allowAdminFields {
		auditRecorder.Record(ctx, fmt.Sprintf("创建用户 %s（ID %d）", user.Account, user.ID))
	} else {
		auditRecorder.Record(utils.WithAuditActor(ctx, user.Account, ""), fmt.Sprintf("注册用户 %s（ID %d）", user.Account, user.ID))
	}

	result, err := s.newUserResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListUsers 按分页和筛选条件查询用户，并返回分页响应 DTO。
func (s *UserService) ListUsers(ctx context.Context, userQuery dto.UserRequest) (*dto.PageResponse[dto.UserResponse], error) {
	page, pageSize, err := utils.NormalizePage(userQuery.Page, userQuery.PageSize)
	if err != nil {
		return nil, err
	}
	if userQuery.Status != nil && !utils.ValidUserStatus(*userQuery.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "用户状态无效")
	}

	userQuery.Page = &page
	userQuery.PageSize = &pageSize
	users, total, err := s.users.List(ctx, userQuery)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "获取用户列表失败", err)
	}

	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	rolesByUserID, err := s.roles.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取用户角色失败")
	}
	responses := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, dto.NewUserResponseWithRoles(user, rolesByUserID[user.ID]))
	}
	return &dto.PageResponse[dto.UserResponse]{
		List: responses, Page: page, PageSize: pageSize, Total: total,
	}, nil
}

// GetUser 获取用户详情。
func (s *UserService) GetUser(ctx context.Context, id uint) (*dto.UserResponse, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, utils.BuildUserRepositoryError(err, "获取用户详情失败")
	}
	result, err := s.newUserResponse(ctx, *user)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *UserService) newUserResponse(ctx context.Context, user model.User) (dto.UserResponse, error) {
	roles, err := s.roles.FindByUserID(ctx, user.ID)
	if err != nil {
		return dto.UserResponse{}, utils.BuildRoleRepositoryError(err, "获取用户角色失败")
	}
	return dto.NewUserResponseWithRoles(user, roles), nil
}

// UpdateUser 更新用户资料。
func (s *UserService) UpdateUser(ctx context.Context, id uint, input dto.UserRequest) (*dto.UserResponse, error) {
	if input.Password != nil || input.OpenID != nil || input.Status != nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "密码、OpenID 和状态不能通过资料接口修改")
	}
	updates := userProfileUpdates(input)
	if len(updates) == 0 {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少提供一个需要修改的字段")
	}

	if account, ok := updates["account"].(string); ok {
		exists, err := s.users.ExistsByAccount(ctx, account, id)
		if err != nil {
			return nil, utils.WrapAppError(utils.KindInternal, "检查用户账号失败", err)
		}
		if exists {
			return nil, utils.NewAppError(utils.KindConflict, "用户账号已存在")
		}
	}

	if err := s.users.UpdateProfile(ctx, id, updates); err != nil {
		return nil, utils.BuildUserRepositoryError(err, "更新用户资料失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改用户 ID %d 的资料", id))
	return s.GetUser(ctx, id)
}

// UpdateUserStatus 修改用户状态。
func (s *UserService) UpdateUserStatus(ctx context.Context, userID uint, input dto.UserRequest) (*dto.UserResponse, error) {
	if input.Status == nil || !utils.ValidUserStatus(*input.Status) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "用户状态无效")
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, utils.BuildUserRepositoryError(err, "获取用户详情失败")
	}
	if err := s.users.UpdateStatus(ctx, userID, *input.Status); err != nil {
		return nil, utils.BuildUserRepositoryError(err, "更新用户状态失败")
	}
	statusName := "正常"
	switch *input.Status {
	case model.UserStatusLocked:
		statusName = "锁定"
	case model.UserStatusCancelled:
		statusName = "注销"
	}
	// 用户状态已成功更新到数据库后立即记录，后续会话撤销失败时仍需保留实际发生的状态变更。
	auditRecorder.Record(ctx, fmt.Sprintf("将用户 %s（ID %d）的状态修改为%s", user.Account, userID, statusName))
	if *input.Status == model.UserStatusLocked || *input.Status == model.UserStatusCancelled {
		if _, err := s.getRedisTemplate().Delete(ctx, user.Account); err != nil {
			return nil, utils.WrapAppError(utils.KindUnavailable, "更新用户状态成功，但撤销登录会话失败", err)
		}
	}
	return s.GetUser(ctx, userID)
}

// CancelUser 将用户状态设置为注销，不物理删除用户记录。
func (s *UserService) CancelUser(ctx context.Context, user utils.SessionData) (*dto.UserResponse, error) {
	status := model.UserStatusCancelled
	return s.UpdateUserStatus(ctx, user.UserID, dto.UserRequest{Status: &status})
}

func userProfileUpdates(input dto.UserRequest) map[string]any {
	updates := make(map[string]any)
	if input.Account != nil {
		updates["account"] = strings.TrimSpace(*input.Account)
	}
	if input.Name != nil {
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Sex != nil {
		updates["sex"] = *input.Sex
	}
	if input.Email != nil {
		updates["email"] = strings.TrimSpace(*input.Email)
	}
	if input.Phone != nil {
		updates["phone"] = strings.TrimSpace(*input.Phone)
	}
	if input.AvatarURL != nil {
		updates["avatarUrl"] = strings.TrimSpace(*input.AvatarURL)
	}
	return updates
}

// ChangePassword 校验旧密码、更新密码哈希并撤销该用户的全部会话。
func (s *UserService) ChangePassword(ctx context.Context, userID uint, input dto.ChangePasswordRequest) error {
	if utf8.RuneCountInString(input.NewPassword) < 8 {
		return utils.NewAppError(utils.KindInvalidArgument, "新密码至少需要 8 个字符")
	}
	if input.OldPassword == input.NewPassword {
		return utils.NewAppError(utils.KindInvalidArgument, "新密码不能与旧密码相同")
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return utils.BuildUserRepositoryError(err, "获取当前用户失败")
	}
	passwordMatches, err := utils.VerifyPassword(user.Password, input.OldPassword)
	if err != nil {
		return utils.WrapAppError(utils.KindInternal, "校验用户密码失败", err)
	}
	if !passwordMatches {
		return utils.NewAppError(utils.KindUnauthenticated, "旧密码错误")
	}

	passwordHash, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		return utils.WrapAppError(utils.KindInvalidArgument, "新密码无效", err)
	}
	if _, err := s.getRedisTemplate().Delete(ctx, user.Account); err != nil {
		return utils.WrapAppError(utils.KindUnavailable, "撤销旧登录会话失败，未更新密码", err)
	}
	if err := s.users.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return utils.BuildUserRepositoryError(err, "更新用户密码失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改用户 %s（ID %d）的密码", user.Account, userID))
	return nil
}
