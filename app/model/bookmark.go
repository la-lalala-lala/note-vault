package model

import "time"

// Bookmark 对应收藏夹表（bookmarks），字段含义与数据库注释保持一致。
// HTTP 请求和响应使用 dto 包中的结构，禁止直接序列化数据库 Model。
type Bookmark struct {
	ID          uint      `gorm:"column:id;primaryKey" json:"-"`                  // 收藏主键
	OwnerID     uint      `gorm:"column:owner_id;not null" json:"-"`              // 所属用户 ID
	Topic       *string   `gorm:"column:topic;size:200" json:"-"`                 // 收藏标题
	Label       *string   `gorm:"column:label;size:100" json:"-"`                 // 收藏标签
	Href        string    `gorm:"column:href;type:text;not null" json:"-"`        // 收藏链接地址
	Icon        *string   `gorm:"column:icon;type:text" json:"-"`                 // 网站图标地址
	Favorite    bool      `gorm:"column:favorite;not null;default:true" json:"-"` // 是否标记为重点收藏
	Description *string   `gorm:"column:description;type:text" json:"-"`          // 收藏描述
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"-"`      // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"`      // 最后更新时间
}

// TableName 明确 Bookmark 使用的数据库表名。
func (Bookmark) TableName() string {
	return "bookmarks"
}
