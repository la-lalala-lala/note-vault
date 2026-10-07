package service

import (
	"context"
	"encoding/json"
	"errors"
	"note-vault/app/config"
	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	authLogoutPermission         = "GET:/api/v1/auth/logout"
	authChangePasswordPermission = "PUT:/api/v1/users/password"
)

// AuthService 负责登录、会话管理、退出和密码修改。
type AuthService struct {
	users *repository.UserRepository
	roles *repository.RoleRepository
	redis *config.RedisTemplate
}

func NewAuthService() *AuthService {
	return &AuthService{
		users: repository.NewUserRepository(),
		roles: repository.NewRoleRepository(),
		redis: nil,
	}
}

func (s *AuthService) getRedisTemplate() *config.RedisTemplate {
	if nil == s.redis {
		s.redis = config.NewRedisTemplate(config.Redis, config.AppConfig.Redis.OperationTimeoutDuration())
	}
	return s.redis
}

// Login 校验账号密码，创建服务端会话并返回原始 Token。
func (s *AuthService) Login(ctx context.Context, input dto.LoginRequest, loginIP string) (*dto.LoginResponse, error) {
	account := strings.TrimSpace(input.Account)
	if account == "" || input.Password == "" {
		return nil, invalidLoginError()
	}

	user, err := s.users.FindByAccount(ctx, account)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, invalidLoginError()
	}
	if err != nil {
		return nil, utils.BuildUserRepositoryError(err, "查询登录用户失败")
	}
	if user.Status != model.UserStatusNormal {
		return nil, invalidLoginError()
	}

	passwordMatches, err := utils.VerifyPassword(user.Password, input.Password)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "校验用户密码失败", err)
	}
	if !passwordMatches {
		return nil, invalidLoginError()
	}

	apiPermissions, menus, superAdmin, err := s.roles.AuthorizationDataByUserID(ctx, user.ID)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取用户权限失败")
	}
	if superAdmin {
		apiPermissions = []string{utils.AllAPIPermissions}
	} else {
		apiPermissions = append(apiPermissions, authLogoutPermission, authChangePasswordPermission)
	}
	userRoles, err := s.roles.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, utils.BuildRoleRepositoryError(err, "获取用户角色失败")
	}
	// 登录前检查清理最大会话数。
	err = s.cleanExcessiveSession(ctx, user.Account)
	if nil != err {
		return nil, err
	}

	token, err := utils.GenerateSessionToken(user.Account, config.AppConfig.Session.TokenBytes)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "生成登录 Token 失败", err)
	}
	tokenHash := utils.HashSessionToken(token)
	expiration := time.Duration(config.AppConfig.Session.ExpireHours) * time.Hour
	session := utils.SessionData{
		UserID:         user.ID,
		Account:        user.Account,
		APIPermissions: apiPermissions,
		LoginIP:        loginIP,
		ExpirationDate: time.Now().Add(expiration),
	}
	serializedSession, err := json.Marshal(session)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "序列化登录会话失败", err)
	}
	if err := s.getRedisTemplate().HSet(ctx, user.Account, tokenHash, serializedSession, expiration); err != nil {
		return nil, utils.WrapAppError(utils.KindUnavailable, "创建登录会话失败", err)
	}

	loginAt := time.Now().UTC()
	if err := s.users.UpdateLoginInfo(ctx, user.ID, loginAt, loginIP); err != nil {
		_, _ = s.getRedisTemplate().HDel(ctx, user.Account, tokenHash)
		return nil, utils.BuildUserRepositoryError(err, "更新最近登录信息失败")
	}

	user.LastLoginAt = &loginAt
	user.LastLoginIP = &loginIP
	menuTree := buildMenuTree(menus)
	if menuTree == nil {
		menuTree = []dto.MenuResponse{}
	}
	auditRecorder.Record(utils.WithAuditActor(ctx, user.Account, loginIP), "登录系统")
	response := dto.LoginResponse{Token: token, User: dto.NewUserResponseWithRoles(*user, userRoles), MenuTree: menuTree}
	return &response, nil
}

// Logout 删除当前登录用户的指定会话。
func (s *AuthService) Logout(ctx context.Context, tokenHash string, account string) error {
	if _, err := s.getRedisTemplate().HDel(ctx, account, tokenHash); err != nil {
		return utils.WrapAppError(utils.KindUnavailable, "退出登录失败", err)
	}
	auditRecorder.Record(ctx, "退出系统")
	return nil
}

// 清理多余的会话
func (s *AuthService) cleanExcessiveSession(ctx context.Context, account string) error {
	// 登录前检查最大会话数。新会话写入后也要计入数量，因此已有会话数达到
	// maxSessions 时，需要先淘汰一个最早到期的会话。
	sessions, err := s.getRedisTemplate().HGetAll(ctx, account)
	if err != nil {
		return utils.WrapAppError(utils.KindUnavailable, "读取已有登录会话失败", err)
	}
	// 待清理的会话数（超过最大允许会话要剔除）
	planCleanSessionCount := len(sessions) - config.AppConfig.Auth.MaxSessions + 1
	if planCleanSessionCount > 0 {
		type sessionEntry struct {
			tokenHash string
			session   utils.SessionData
		}

		sessionEntries := make([]sessionEntry, 0, len(sessions))
		for tokenHash, rawSession := range sessions {
			var sessionData utils.SessionData
			if err := json.Unmarshal([]byte(rawSession), &sessionData); err != nil {
				return utils.WrapAppError(utils.KindUnavailable, "解析已有登录会话失败", err)
			}
			sessionEntries = append(sessionEntries, sessionEntry{
				tokenHash: tokenHash,
				session:   sessionData,
			})
		}

		sort.Slice(sessionEntries, func(i, j int) bool {
			left := sessionEntries[i]
			right := sessionEntries[j]
			if left.session.ExpirationDate.Equal(right.session.ExpirationDate) {
				return left.tokenHash < right.tokenHash
			}
			return left.session.ExpirationDate.Before(right.session.ExpirationDate)
		})
		// 过期的会话
		fieldsToDelete := make([]string, 0, planCleanSessionCount)
		for index := 0; index < planCleanSessionCount; index++ {
			fieldsToDelete = append(fieldsToDelete, sessionEntries[index].tokenHash)
		}
		if _, err := s.getRedisTemplate().HDel(ctx, account, fieldsToDelete...); err != nil {
			return utils.WrapAppError(utils.KindUnavailable, "清理超出数量的登录会话失败", err)
		}
	}
	return nil
}

func invalidLoginError() error {
	return utils.NewAppError(utils.KindUnauthenticated, "账号或密码错误，或用户状态不可登录")
}
