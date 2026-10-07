package controller

import (
	"note-vault/app/dto"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var auditService = service.NewAuditService()

// ListAuditLogs 查询审计日志，不提供新增、修改和删除入口。
func ListAuditLogs(c *gin.Context) {
	var input dto.AuditLogRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := auditService.List(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}
