package model

import "time"

// AuditLog 对应审计日志表；日志只允许追加和查询。
type AuditLog struct {
	ID              uint      `gorm:"column:id;primaryKey" json:"-"`                         // 日志主键
	OperatorAccount string    `gorm:"column:operator_account;not null" json:"-"`             // 操作发生时的账号快照
	Operation       string    `gorm:"column:operation;not null" json:"-"`                    // 操作内容
	IP              *string   `gorm:"column:ip" json:"-"`                                    // 操作来源 IP
	Location        *string   `gorm:"column:location" json:"-"`                              // 操作发生地点
	OperatedAt      time.Time `gorm:"column:operated_at;default:CURRENT_TIMESTAMP" json:"-"` // 数据库生成的操作时间
}

// TableName 明确审计日志使用的数据库表名。
func (AuditLog) TableName() string { return "audit_logs" }
