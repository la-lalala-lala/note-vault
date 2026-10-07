package dto

import (
	"time"

	"note-vault/app/model"
)

// NotebookRequest 是笔记簿新增、修改、查询、排序和状态修改共用的请求结构。
// OwnerID 不由客户端提交，所有笔记簿操作统一使用认证上下文中的当前用户 ID。
type NotebookRequest struct {
	Name        *string `json:"name" form:"name" binding:"omitempty,max=100"`
	Icon        *string `json:"icon" form:"icon" binding:"omitempty,max=100"`
	Tag         *string `json:"tag" form:"tag" binding:"omitempty,max=100"`
	Description *string `json:"description" form:"description" binding:"omitempty,max=1000"`
	Status      *int16  `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
	SortOrder   *int    `json:"sortOrder" form:"sortOrder" binding:"omitempty,min=0"`
	Page        *int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize    *int    `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// NotebookResponse 是笔记簿对外返回结构，不暴露 OwnerID 和删除字段。
type NotebookResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Tag         *string   `json:"tag,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      int16     `json:"status"`
	SortOrder   int       `json:"sortOrder"`
	NoteCount   int64     `json:"noteCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NewNotebookResponse 将数据库 Model 转换为不包含数据归属字段的响应 DTO。
func NewNotebookResponse(notebook model.Notebook) NotebookResponse {
	return NotebookResponse{
		ID: notebook.ID, Name: notebook.Name, Icon: notebook.Icon, Tag: notebook.Tag, Description: notebook.Description,
		Status: notebook.Status, SortOrder: notebook.SortOrder,
		CreatedAt: notebook.CreatedAt, UpdatedAt: notebook.UpdatedAt,
	}
}

// NewNotebookResponses 批量转换笔记簿响应 DTO。
func NewNotebookResponses(notebooks []model.Notebook, counts map[uint]int64) []NotebookResponse {
	responses := make([]NotebookResponse, 0, len(notebooks))
	for _, notebook := range notebooks {
		response := NewNotebookResponse(notebook)
		response.NoteCount = counts[notebook.ID]
		responses = append(responses, response)
	}
	return responses
}
