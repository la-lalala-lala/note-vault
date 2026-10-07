package dto

import (
	"time"

	"note-vault/app/model"
)

// DraftRequest 是草稿自动保存、发布和分页查询共用的请求结构。
// 自动保存不要求客户端提交草稿版本号；提交已有笔记的草稿时，
// NoteVersion 用于校验正式笔记的并发修改版本。
type DraftRequest struct {
	NoteID      *uint   `json:"noteId" form:"noteId" binding:"omitempty,min=1"`
	NoteVersion *uint   `json:"noteVersion" form:"noteVersion" binding:"omitempty,min=1"`
	NotebookID  *uint   `json:"notebookId" form:"notebookId" binding:"omitempty,min=1"`
	Title       *string `json:"title" form:"title" binding:"omitempty,max=200"`
	Label       *string `json:"label" form:"label" binding:"omitempty,max=200"`
	Content     *string `json:"content" form:"content" binding:"omitempty,max=10485760"`
	ContentType *string `json:"contentType" form:"contentType" binding:"omitempty,max=50"`
	Page        *int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize    *int    `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// DraftResponse 是草稿对外返回结构，不暴露 OwnerID。
// 列表接口不选择 Content，正文仅在详情及写操作响应中返回。
// Version 是服务端保存修订号，仅用于展示和排查，不要求客户端回传。
type DraftResponse struct {
	ID          uint       `json:"id"`
	DraftKey    string     `json:"draftKey"`
	NoteID      *uint      `json:"noteId,omitempty"`
	NotebookID  *uint      `json:"notebookId,omitempty"`
	Title       string     `json:"title"`
	Label       *string    `json:"label,omitempty"`
	Content     *string    `json:"content,omitempty"`
	ContentType string     `json:"contentType"`
	Version     uint       `json:"version"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// NewDraftResponse 将数据库 Model 转换为不包含数据归属字段的响应 DTO。
func NewDraftResponse(draft model.Draft) DraftResponse {
	return DraftResponse{
		ID: draft.ID, DraftKey: draft.DraftKey, NoteID: draft.NoteID,
		NotebookID: draft.NotebookID,
		Title:      draft.Title, Label: draft.Label, Content: draft.Content,
		ContentType: draft.ContentType,
		Version:     draft.Version, ExpiresAt: draft.ExpiresAt,
		CreatedAt: draft.CreatedAt, UpdatedAt: draft.UpdatedAt,
	}
}

// NewDraftResponses 批量转换草稿响应 DTO。
func NewDraftResponses(drafts []model.Draft) []DraftResponse {
	responses := make([]DraftResponse, 0, len(drafts))
	for _, draft := range drafts {
		responses = append(responses, NewDraftResponse(draft))
	}
	return responses
}
