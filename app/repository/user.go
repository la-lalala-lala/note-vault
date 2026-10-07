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

// db 为可选注入的 GORM 数据库连接。未注入时使用 config.DB，兼容现有全局
// 初始化方式；注入其他 GORM Dialector 创建的连接即可复用本 Repository。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储。
func NewUserRepository(db ...*gorm.DB) *UserRepository {
	repository := &UserRepository{}
	if len(db) > 0 {
		repository.db = db[0]
	}
	return repository
}

func (r *UserRepository) database() (*gorm.DB, error) {
	if r.db != nil {
		return r.db, nil
	}
	if config.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	return config.DB, nil
}

// CreateWithRole 在同一事务中创建用户并绑定默认角色。
func (r *UserRepository) CreateWithRole(ctx context.Context, user *model.User, roleCode string) error {
	db, err := r.database()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := typed.G[model.User](tx).Create(ctx, user); err != nil {
			return err
		}
		if roleCode == "" {
			return nil
		}

		role, err := typed.G[model.Role](tx).
			Where(generated.Role.Code.Eq(roleCode)).
			Take(ctx)
		if err != nil {
			return err
		}

		userRole := model.UserRole{UserID: user.ID, RoleID: role.ID}
		return typed.G[model.UserRole](tx).Create(ctx, &userRole)
	})
}

// ExistsByAccount 判断账号是否已被其他用户使用，比较时忽略大小写。
func (r *UserRepository) ExistsByAccount(ctx context.Context, account string, excludeUserID uint) (bool, error) {
	db, err := r.database()
	if err != nil {
		return false, err
	}

	query := typed.G[model.User](db).Where(
		generated.User.Account.Expr("LOWER(?) = LOWER(?)", generated.User.Account.Column(), strings.TrimSpace(account)),
	)
	if excludeUserID > 0 {
		query = query.Where(generated.User.ID.Neq(excludeUserID))
	}

	count, err := query.Count(ctx, "*")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByAccount 按大小写不敏感的账号查询用户。
func (r *UserRepository) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	user, err := typed.G[model.User](db).Where(
		generated.User.Account.Expr("LOWER(?) = LOWER(?)", generated.User.Account.Column(), strings.TrimSpace(account)),
	).Take(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 查询用户详情，敏感字段由 Service 转换响应 DTO 时隔离。
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	db, err := r.database()
	if err != nil {
		return nil, err
	}

	user, err := typed.G[model.User](db).Where(generated.User.ID.Eq(id)).Take(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List 按筛选条件分页查询用户列表，字段由 GORM Model 和生成字段辅助层维护。
func (r *UserRepository) List(ctx context.Context, userQuery dto.UserRequest) ([]model.User, int64, error) {
	db, err := r.database()
	if err != nil {
		return nil, 0, err
	}

	usersQuery := typed.G[model.User](db).Scopes()
	if userQuery.Account != nil && strings.TrimSpace(*userQuery.Account) != "" {
		pattern := "%" + strings.TrimSpace(*userQuery.Account) + "%"
		usersQuery = usersQuery.Where(generated.User.Account.Expr("LOWER(?) LIKE LOWER(?)", generated.User.Account.Column(), pattern))
	}
	if userQuery.Name != nil && strings.TrimSpace(*userQuery.Name) != "" {
		usersQuery = usersQuery.Where(generated.User.Name.Like("%" + strings.TrimSpace(*userQuery.Name) + "%"))
	}
	if userQuery.Email != nil && strings.TrimSpace(*userQuery.Email) != "" {
		usersQuery = usersQuery.Where(generated.User.Email.Like("%" + strings.TrimSpace(*userQuery.Email) + "%"))
	}
	if userQuery.Phone != nil && strings.TrimSpace(*userQuery.Phone) != "" {
		usersQuery = usersQuery.Where(generated.User.Phone.Like("%" + strings.TrimSpace(*userQuery.Phone) + "%"))
	}
	if userQuery.Status != nil {
		usersQuery = usersQuery.Where(generated.User.Status.Eq(*userQuery.Status))
	}

	total, err := usersQuery.Count(ctx, "*")
	if err != nil {
		return nil, 0, err
	}

	offset := (*userQuery.Page - 1) * *userQuery.PageSize
	users, err := usersQuery.
		Order(generated.User.ID.OrderExpr("? DESC", generated.User.ID)).
		Offset(offset).
		Limit(*userQuery.PageSize).
		Find(ctx)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// UpdateProfile 更新允许修改的用户资料字段。
func (r *UserRepository) UpdateProfile(ctx context.Context, id uint, updates map[string]any) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	assignments, err := profileAssignments(updates)
	if err != nil {
		return err
	}
	_, err = typed.G[model.User](db).Where(generated.User.ID.Eq(id)).Set(assignments...).Update(ctx)
	return err
}

// UpdateLoginInfo 更新用户最近登录时间和 IP。
func (r *UserRepository) UpdateLoginInfo(ctx context.Context, id uint, loginAt time.Time, loginIP string) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	_, err = typed.G[model.User](db).Where(generated.User.ID.Eq(id)).Set(
		generated.User.LastLoginAt.Set(loginAt),
		generated.User.LastLoginIP.Set(loginIP),
	).Update(ctx)
	return err
}

// UpdatePassword 更新用户密码哈希。
func (r *UserRepository) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	_, err = typed.G[model.User](db).Where(generated.User.ID.Eq(id)).Set(
		generated.User.Password.Set(passwordHash),
	).Update(ctx)
	return err
}

// UpdateStatus 修改用户状态。
func (r *UserRepository) UpdateStatus(ctx context.Context, id uint, status int16) error {
	db, err := r.database()
	if err != nil {
		return err
	}
	_, err = typed.G[model.User](db).Where(generated.User.ID.Eq(id)).Set(generated.User.Status.Set(status)).Update(ctx)
	return err
}

func profileAssignments(updates map[string]any) ([]clause.Assigner, error) {
	assignments := make([]clause.Assigner, 0, len(updates))
	for name, value := range updates {
		switch name {
		case "account":
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("invalid account update")
			}
			assignments = append(assignments, generated.User.Account.Set(text))
		case "name":
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("invalid name update")
			}
			assignments = append(assignments, generated.User.Name.Set(text))
		case "sex":
			sex, ok := value.(int16)
			if !ok {
				return nil, errors.New("invalid sex update")
			}
			assignments = append(assignments, generated.User.Sex.Set(sex))
		case "email":
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("invalid email update")
			}
			assignments = append(assignments, generated.User.Email.Set(text))
		case "phone":
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("invalid phone update")
			}
			assignments = append(assignments, generated.User.Phone.Set(text))
		case "avatarUrl":
			text, ok := value.(string)
			if !ok {
				return nil, errors.New("invalid avatar update")
			}
			assignments = append(assignments, generated.User.AvatarURL.Set(text))
		default:
			return nil, errors.New("unsupported user profile field")
		}
	}
	return assignments, nil
}
