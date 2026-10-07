package model

import "time"

// UserRole 对应用户角色关联表（user_roles），字段含义与数据库注释保持一致。
type UserRole struct {
	UserID    uint      `gorm:"column:user_id;primaryKey" json:"-"`        // 用户 ID
	RoleID    uint      `gorm:"column:role_id;primaryKey" json:"-"`        // 角色 ID
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"-"` // 授权时间
}

// TableName 明确 UserRole 使用的数据库表名。
func (UserRole) TableName() string {
	return "user_roles"
}
