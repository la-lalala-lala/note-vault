package dto

import (
	"time"

	"note-vault/app/model"
)

// NoteRequest 是笔记新增、修改、移动和分页查询共用的请求结构。
// OwnerID 和 Status 不由客户端提交，普通笔记接口只操作当前用户的正常笔记。
type NoteRequest struct {
	NotebookID  *uint      `json:"notebookId" form:"notebookId" binding:"omitempty,min=1"`
	Title       *string    `json:"title" form:"title" binding:"omitempty,max=200"`
	Label       *string    `json:"label" form:"label" binding:"omitempty,max=200"`
	Summary     *string    `json:"summary" form:"summary" binding:"omitempty,max=2000"`
	CoverURL    *string    `json:"coverUrl" form:"coverUrl" binding:"omitempty,max=500"`
	Content     *string    `json:"content" form:"content" binding:"omitempty,max=10485760"`
	ContentType *string    `json:"contentType" form:"contentType" binding:"omitempty,max=50"`
	Version     *uint      `json:"version" form:"version" binding:"omitempty,min=1"`
	UpdatedFrom *time.Time `json:"updatedFrom" form:"updatedFrom" time_format:"2006-01-02T15:04:05Z07:00"`
	UpdatedTo   *time.Time `json:"updatedTo" form:"updatedTo" time_format:"2006-01-02T15:04:05Z07:00"`
	Page        *int       `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize    *int       `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// NoteResponse 是笔记对外返回结构，不暴露 OwnerID 和 DeletedBy。
// 分页列表不查询 Content 和 ContentURI，因此正文只会在详情及写操作响应中返回。
type NoteResponse struct {
	ID                     uint       `json:"id"`
	NotebookID             uint       `json:"notebookId"`
	Title                  string     `json:"title"`
	Label                  *string    `json:"label,omitempty"`
	Summary                *string    `json:"summary,omitempty"`
	CoverURL               *string    `json:"coverUrl,omitempty"`
	Content                *string    `json:"content,omitempty"`
	ContentURI             *string    `json:"contentUri,omitempty"`
	ContentType            string     `json:"contentType"`
	Status                 int16      `json:"status"`
	Version                uint       `json:"version"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	DeletedAt              *time.Time `json:"deletedAt,omitempty"`
	PurgeAt                *time.Time `json:"purgeAt,omitempty"`
	RemainingRetentionDays *int       `json:"remainingRetentionDays,omitempty"`
}

// NewNoteResponse 将数据库 Model 转换为不包含数据归属字段的响应 DTO。
func NewNoteResponse(note model.Note) NoteResponse {
	return NoteResponse{
		ID: note.ID, NotebookID: note.NotebookID, Title: note.Title,
		Label: note.Label, Summary: note.Summary, CoverURL: note.CoverURL,
		Content: note.Content, ContentURI: note.ContentURI, ContentType: note.ContentType,
		Status: note.Status, Version: note.Version,
		CreatedAt: note.CreatedAt, UpdatedAt: note.UpdatedAt,
		DeletedAt: note.DeletedAt, PurgeAt: note.PurgeAt,
	}
}

// NewNoteResponses 批量转换笔记响应 DTO。
func NewNoteResponses(notes []model.Note) []NoteResponse {
	responses := make([]NoteResponse, 0, len(notes))
	for _, note := range notes {
		responses = append(responses, NewNoteResponse(note))
	}
	return responses
}
