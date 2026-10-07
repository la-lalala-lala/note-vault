package dto

import (
	"time"

	"note-vault/app/model"
)

// BookmarkRequest 是收藏夹新增、修改和分页查询共用的请求结构。
// OwnerID 不由客户端提交，所有操作统一使用认证上下文中的当前用户 ID。
type BookmarkRequest struct {
	Topic       *string    `json:"topic" form:"topic" binding:"omitempty,max=200"`
	Label       *string    `json:"label" form:"label" binding:"omitempty,max=100"`
	Href        *string    `json:"href" form:"href" binding:"omitempty,max=2048"`
	Icon        *string    `json:"icon" form:"icon" binding:"omitempty,max=1000"`
	Favorite    *bool      `json:"favorite" form:"favorite"`
	Description *string    `json:"description" form:"description" binding:"omitempty,max=2000"`
	CreatedFrom *time.Time `json:"createdFrom" form:"createdFrom" time_format:"2006-01-02T15:04:05Z07:00"`
	CreatedTo   *time.Time `json:"createdTo" form:"createdTo" time_format:"2006-01-02T15:04:05Z07:00"`
	Page        *int       `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize    *int       `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// BookmarkResponse 是收藏夹对外返回结构，不暴露 OwnerID。
type BookmarkResponse struct {
	ID          uint      `json:"id"`
	Topic       *string   `json:"topic,omitempty"`
	Label       *string   `json:"label,omitempty"`
	Href        string    `json:"href"`
	Icon        *string   `json:"icon,omitempty"`
	Favorite    bool      `json:"favorite"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// NewBookmarkResponse 将数据库 Model 转换为不包含数据归属字段的响应 DTO。
func NewBookmarkResponse(bookmark model.Bookmark) BookmarkResponse {
	return BookmarkResponse{
		ID: bookmark.ID, Topic: bookmark.Topic, Label: bookmark.Label, Href: bookmark.Href,
		Icon: bookmark.Icon, Favorite: bookmark.Favorite, Description: bookmark.Description,
		CreatedAt: bookmark.CreatedAt, UpdatedAt: bookmark.UpdatedAt,
	}
}

// NewBookmarkResponses 批量转换收藏夹响应 DTO。
func NewBookmarkResponses(bookmarks []model.Bookmark) []BookmarkResponse {
	responses := make([]BookmarkResponse, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		responses = append(responses, NewBookmarkResponse(bookmark))
	}
	return responses
}
