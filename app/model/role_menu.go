package model

import "time"

// RoleMenu 对应角色菜单权限关联表（role_menus），字段含义与数据库注释保持一致。
type RoleMenu struct {
	RoleID    uint      `gorm:"column:role_id;primaryKey" json:"-"`        // 角色 ID
	MenuID    uint      `gorm:"column:menu_id;primaryKey" json:"-"`        // 菜单或权限资源 ID
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"-"` // 授权时间
}

// TableName 明确 RoleMenu 使用的数据库表名。
func (RoleMenu) TableName() string {
	return "role_menus"
}
