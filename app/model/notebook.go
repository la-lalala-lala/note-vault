package model

import "time"

const (
	NotebookStatusNormal   int16 = 1
	NotebookStatusDisabled int16 = 2
	NotebookStatusDeleted  int16 = 3
)

// Notebook 对应笔记簿表（notebooks），字段含义与数据库注释保持一致。
type Notebook struct {
	ID          uint       `gorm:"column:id;primaryKey" json:"-"`                     // 笔记簿主键
	OwnerID     uint       `gorm:"column:owner_id;not null" json:"-"`                 // 所属用户 ID
	Name        string     `gorm:"column:name;size:100;not null" json:"-"`            // 笔记簿名
	Icon        string     `gorm:"column:icon;size:100;not null;default:''" json:"-"` // 封面图标名
	Tag         *string    `gorm:"column:tag;size:100" json:"-"`                      // 笔记簿标签
	Description *string    `gorm:"column:description;type:text" json:"-"`             // 笔记簿描述
	Status      int16      `gorm:"column:status;not null;default:1" json:"-"`         // 笔记簿状态：1 正常，2 禁用，3 删除
	SortOrder   int        `gorm:"column:sort_order;not null;default:0" json:"-"`     // 笔记簿排序值
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`         // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"-"`         // 最后更新时间
	DeletedAt   *time.Time `gorm:"column:deleted_at" json:"-"`                        // 删除时间
}

// TableName 明确 Notebook 使用的数据库表名。
func (Notebook) TableName() string {
	return "notebooks"
}
