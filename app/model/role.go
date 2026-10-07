package model

import "time"

const (
	RoleStatusEnabled  int16 = 1
	RoleStatusDisabled int16 = 2
	RoleCodeSuperAdmin       = "SUPER_ADMIN"
	RoleCodeUser             = "USER"
)

// Role 对应角色表（roles），字段含义与数据库注释保持一致。
type Role struct {
	ID          uint      `gorm:"column:id;primaryKey" json:"-"`             // 角色主键
	Name        string    `gorm:"column:name;size:100;not null" json:"-"`    // 角色名称
	Code        string    `gorm:"column:code;size:100;not null" json:"-"`    // 角色编码
	Description *string   `gorm:"column:description;size:500" json:"-"`      // 角色描述
	Status      int16     `gorm:"column:status;not null;default:1" json:"-"` // 角色状态：1 启用，2 禁用
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"-"` // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"` // 最后更新时间
}

// TableName 明确 Role 使用的数据库表名。
func (Role) TableName() string {
	return "roles"
}
