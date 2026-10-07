package model

import "time"

const (
	NoteStatusNormal int16 = 1
	NoteStatusTrash  int16 = 2
)

// Note 对应笔记表（notes），字段含义与数据库注释保持一致。
type Note struct {
	ID          uint       `gorm:"column:id;primaryKey" json:"-"`                 // 笔记主键
	OwnerID     uint       `gorm:"column:owner_id;not null" json:"-"`             // 所属用户 ID
	NotebookID  uint       `gorm:"column:notebook_id;not null" json:"-"`          // 所属笔记簿 ID
	Title       string     `gorm:"column:title;size:200;not null" json:"-"`       // 笔记标题
	Label       *string    `gorm:"column:label;size:200" json:"-"`                // 笔记标签
	Summary     *string    `gorm:"column:summary;type:text" json:"-"`             // 笔记摘要
	CoverURL    *string    `gorm:"column:cover_url;size:500" json:"-"`            // 笔记封面地址
	Content     *string    `gorm:"column:content;type:text" json:"-"`             // 笔记正文内容
	ContentURI  *string    `gorm:"column:content_uri;size:1000" json:"-"`         // 正文外部存储地址
	ContentType string     `gorm:"column:content_type;size:50;not null" json:"-"` // 正文格式，例如 markdown 或 html
	Status      int16      `gorm:"column:status;not null;default:1" json:"-"`     // 笔记状态：1 正常，2 回收站
	Version     uint       `gorm:"column:version;not null;default:1" json:"-"`    // 乐观锁版本号
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"-"`                    // 进入回收站时间
	DeletedBy   *uint      `gorm:"column:deleted_by" json:"-"`                    // 执行删除的用户 ID
	PurgeAt     *time.Time `gorm:"column:purge_at" json:"-"`                      // 计划永久清理时间
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`     // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`     // 最后更新时间
}

// TableName 明确 Note 使用的数据库表名。
func (Note) TableName() string {
	return "notes"
}
