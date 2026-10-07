package dto

import (
	"time"

	"note-vault/app/model"
)

// FileRequest 是文件列表查询共用的请求结构。
// 文件上传使用 multipart/form-data 的 file 字段，不通过该结构接收文件内容。
type FileRequest struct {
	FileName *string `json:"fileName" form:"fileName" binding:"omitempty,max=255"`
	FileType *string `json:"fileType" form:"fileType" binding:"omitempty,max=100"`
	Page     *int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize *int    `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// Base64ImageRequest 是 Base64 图片上传请求；Base64 可传纯编码或 data URL。
type Base64ImageRequest struct {
	FileName string `json:"fileName" binding:"required,max=255"`
	Base64   string `json:"base64" binding:"required"`
}

// FileResponse 是文件元数据对外返回结构，不暴露 OwnerID、ObjectKey、Status 和 DeletedAt。
type FileResponse struct {
	ID        uint      `json:"id"`
	FileName  string    `json:"fileName"`
	FileURL   *string   `json:"fileUrl,omitempty"`
	FileType  *string   `json:"fileType,omitempty"`
	FileSize  int64     `json:"fileSize"`
	Checksum  *string   `json:"checksum,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewFileResponse 将数据库 Model 转换为文件元数据响应 DTO。
func NewFileResponse(file model.File) FileResponse {
	return FileResponse{
		ID: file.ID, FileName: file.FileName, FileURL: file.FileURL,
		FileType: file.FileType, FileSize: file.FileSize, Checksum: file.Checksum,
		CreatedAt: file.CreatedAt, UpdatedAt: file.UpdatedAt,
	}
}

// NewFileResponses 批量转换文件元数据响应 DTO。
func NewFileResponses(files []model.File) []FileResponse {
	responses := make([]FileResponse, 0, len(files))
	for _, file := range files {
		responses = append(responses, NewFileResponse(file))
	}
	return responses
}
