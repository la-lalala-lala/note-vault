package model

import "time"

const (
	MenuTypeDirectory int16 = 1
	MenuTypePage      int16 = 2
	MenuTypeButton    int16 = 3
	MenuTypeAPI       int16 = 4

	MenuStatusEnabled  int16 = 1
	MenuStatusDisabled int16 = 2
)

// Menu 对应菜单和权限资源表（menus），字段含义与数据库注释保持一致。
type Menu struct {
	ID         uint      `gorm:"column:id;primaryKey" json:"-"`                 // 菜单权限主键
	ParentID   *uint     `gorm:"column:parent_id" json:"-"`                     // 父级菜单 ID
	Name       string    `gorm:"column:name;size:100;not null" json:"-"`        // 菜单或权限名称
	Code       string    `gorm:"column:code;size:150;not null" json:"-"`        // 权限编码
	Type       int16     `gorm:"column:type;not null" json:"-"`                 // 资源类型：1 目录，2 页面，3 按钮，4 API
	RoutePath  *string   `gorm:"column:route_path;size:300" json:"-"`           // 前端路由地址
	Component  *string   `gorm:"column:component;size:300" json:"-"`            // 前端组件路径
	HTTPMethod *string   `gorm:"column:http_method;size:10" json:"-"`           // 接口 HTTP 方法，API 类型资源必填
	APIPath    *string   `gorm:"column:api_path;size:500" json:"-"`             // Gin 路由模板，API 类型资源必填，例如 /api/v1/users/:id
	SortOrder  int       `gorm:"column:sort_order;not null;default:0" json:"-"` // 同级排序值
	Visible    bool      `gorm:"column:visible;not null;default:true" json:"-"` // 是否在菜单中显示
	Status     int16     `gorm:"column:status;not null;default:1" json:"-"`     // 资源状态：1 启用，2 禁用
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"-"`     // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"`     // 最后更新时间
}

// TableName 明确 Menu 使用的数据库表名。
func (Menu) TableName() string {
	return "menus"
}
