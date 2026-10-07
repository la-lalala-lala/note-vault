package service

import (
	"context"
	"log"
	"net"
	"strings"

	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
)

type AuditService struct {
	logs *repository.AuditLogRepository
}

func NewAuditService() *AuditService { return &AuditService{logs: repository.NewAuditLogRepository()} }

// Record 在业务写成功后记录账号快照和操作；审计存储失败需记录系统日志，不改变已提交的业务结果。
func (s *AuditService) Record(ctx context.Context, operation string) {
	actor, ok := utils.CurrentAuditActor(ctx)
	if !ok {
		log.Printf("audit_error operation=%q error=missing authenticated actor", operation)
		return
	}
	entry := model.AuditLog{OperatorAccount: actor.Account, Operation: operation}
	if parsed := net.ParseIP(actor.IP); parsed != nil {
		ip := parsed.String()
		entry.IP = &ip
		if parsed.IsLoopback() || parsed.IsPrivate() {
			location := "局域网"
			entry.Location = &location
		}
	}
	if err := s.logs.Create(ctx, &entry); err != nil {
		log.Printf("audit_error account=%q operation=%q error=%v", actor.Account, operation, err)
	}
}

func (s *AuditService) List(ctx context.Context, input dto.AuditLogRequest) (*dto.PageResponse[dto.AuditLogResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	if input.OperatedFrom != nil && input.OperatedTo != nil && input.OperatedFrom.After(*input.OperatedTo) {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "操作时间起始值不能晚于结束值")
	}
	if input.IP != nil && strings.TrimSpace(*input.IP) != "" {
		if parsed := net.ParseIP(strings.TrimSpace(*input.IP)); parsed == nil {
			return nil, utils.NewAppError(utils.KindInvalidArgument, "IP 地址无效")
		} else {
			ip := parsed.String()
			input.IP = &ip
		}
	}
	input.Page, input.PageSize = &page, &pageSize
	logs, total, err := s.logs.List(ctx, input)
	if err != nil {
		return nil, utils.WrapAppError(utils.KindInternal, "查询审计日志失败", err)
	}
	return &dto.PageResponse[dto.AuditLogResponse]{List: dto.NewAuditLogResponses(logs), Page: page, PageSize: pageSize, Total: total}, nil
}

var auditRecorder = NewAuditService()
