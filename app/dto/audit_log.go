package dto

import (
	"time"

	"note-vault/app/model"
)

// AuditLogRequest 是审计日志查询条件；审计日志不提供写接口。
type AuditLogRequest struct {
	OperatorAccount *string    `form:"operatorAccount" binding:"omitempty,max=50"`
	Operation       *string    `form:"operation" binding:"omitempty,max=200"`
	IP              *string    `form:"ip"`
	OperatedFrom    *time.Time `form:"operatedFrom" time_format:"2006-01-02T15:04:05Z07:00"`
	OperatedTo      *time.Time `form:"operatedTo" time_format:"2006-01-02T15:04:05Z07:00"`
	Page            *int       `form:"page" binding:"omitempty,min=1"`
	PageSize        *int       `form:"pageSize" binding:"omitempty,min=1,max=100"`
}

type AuditLogResponse struct {
	ID              uint      `json:"id"`
	OperatorAccount string    `json:"operatorAccount"`
	Operation       string    `json:"operation"`
	IP              *string   `json:"ip"`
	Location        *string   `json:"location"`
	OperatedAt      time.Time `json:"operatedAt"`
}

func NewAuditLogResponses(logs []model.AuditLog) []AuditLogResponse {
	result := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, AuditLogResponse{ID: log.ID, OperatorAccount: log.OperatorAccount,
			Operation: log.Operation, IP: log.IP, Location: log.Location, OperatedAt: log.OperatedAt})
	}
	return result
}
