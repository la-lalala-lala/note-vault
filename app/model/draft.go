package model

import "time"

// Draft 对应笔记草稿表（drafts），字段含义与数据库注释保持一致。
type Draft struct {
	ID          uint       `gorm:"column:id;primaryKey" json:"-"`                                  // 草稿主键
	OwnerID     uint       `gorm:"column:owner_id;not null" json:"-"`                              // 所属用户 ID
	DraftKey    string     `gorm:"column:draft_key;size:100;not null" json:"-"`                    // 客户端草稿业务标识，用于幂等保存
	NoteID      *uint      `gorm:"column:note_id" json:"-"`                                        // 对应的正式笔记 ID，新建笔记时为空
	NotebookID  *uint      `gorm:"column:notebook_id" json:"-"`                                    // 目标笔记簿 ID
	Title       string     `gorm:"column:title;size:200;not null;default:''" json:"-"`             // 草稿标题
	Label       *string    `gorm:"column:label;size:200" json:"-"`                                 // 草稿标签
	Content     *string    `gorm:"column:content;type:text" json:"-"`                              // 草稿正文内容
	ContentURI  *string    `gorm:"column:content_uri;size:1000" json:"-"`                          // 草稿正文外部存储地址
	ContentType string     `gorm:"column:content_type;size:50;not null;default:markdown" json:"-"` // 正文格式，例如 markdown 或 html
	Version     uint       `gorm:"column:version;not null;default:1" json:"-"`                     // 服务端草稿保存修订号，不作为客户端并发控制字段
	ExpiresAt   *time.Time `gorm:"column:expires_at" json:"-"`                                     // 草稿过期时间
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`                      // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`                      // 最后更新时间
}

// TableName 明确 Draft 使用的数据库表名。
func (Draft) TableName() string {
	return "drafts"
}
