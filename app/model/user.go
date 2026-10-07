package model

import "time"

const (
	UserStatusNormal    int16 = 1
	UserStatusLocked    int16 = 2
	UserStatusCancelled int16 = 3

	LoginMethodPassword = "PASSWORD"
)

// User 对应用户表（users），字段含义与数据库注释保持一致。
// HTTP 请求和响应必须使用 dto 包中的独立结构，禁止直接序列化数据库 Model。
type User struct {
	ID uint `gorm:"column:id;primaryKey" json:"-"` // 用户主键

	Account  string `gorm:"column:account;size:50;not null" json:"-"`   // 登录账号
	Name     string `gorm:"column:name;size:100;not null" json:"-"`     // 用户名称
	Password string `gorm:"column:password;size:255;not null" json:"-"` // 用户密码，仅保存密码哈希值，禁止保存明文

	OpenID    *string `gorm:"column:open_id;size:100" json:"-"`    // 第三方平台 OpenID
	Sex       *int16  `gorm:"column:sex" json:"-"`                 // 性别：0 未知，1 男，2 女
	Email     *string `gorm:"column:email;size:150" json:"-"`      // 电子邮箱
	Phone     *string `gorm:"column:phone;size:30" json:"-"`       // 手机号码
	AvatarURL *string `gorm:"column:avatar_url;size:500" json:"-"` // 头像地址

	Status      int16      `gorm:"column:status;not null;default:1" json:"-"`                      // 用户状态：1 正常，2 锁定，3 注销
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"-"`                                  // 最近登录时间
	LastLoginIP *string    `gorm:"column:last_login_ip;type:inet" json:"-"`                        // 最近登录 IP
	LoginMethod string     `gorm:"column:login_method;size:32;not null;default:PASSWORD" json:"-"` // 登录方式
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`                      // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`                      // 最后更新时间
}

// TableName 明确 User 使用的数据库表名。
func (User) TableName() string {
	return "users"
}
