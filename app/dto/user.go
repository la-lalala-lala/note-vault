package dto

import (
	"time"

	"note-vault/app/model"
)

// UserRequest 是用户写操作共用的输入结构。
//
// 注册、新增、资料修改和状态修改都使用这一个结构，具体哪些字段必填、
// 哪些字段允许修改由 Service 按业务场景校验，避免为每个接口复制一份用户字段。
type UserRequest struct {
	Account   *string `json:"account" form:"account" binding:"omitempty,min=3,max=50"`
	Name      *string `json:"name" form:"name" binding:"omitempty,min=1,max=100"`
	Password  *string `json:"password" form:"password" binding:"omitempty,min=8,max=72"`
	OpenID    *string `json:"openId" form:"openId" binding:"omitempty,max=100"`
	Sex       *int16  `json:"sex" form:"sex" binding:"omitempty,oneof=0 1 2"`
	Email     *string `json:"email" form:"email" binding:"omitempty,email,max=150"`
	Phone     *string `json:"phone" form:"phone" binding:"omitempty,max=30"`
	AvatarURL *string `json:"avatarUrl" form:"avatarUrl" binding:"omitempty,max=500"`
	Status    *int16  `json:"status" form:"status" binding:"omitempty,oneof=1 2 3"`
	Page      *int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize  *int    `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// UserResponse 是用户对外响应结构，故意不包含 Password 字段。
type UserResponse struct {
	ID          uint           `json:"id"`
	Account     string         `json:"account"`
	Name        string         `json:"name"`
	Roles       []RoleResponse `json:"roles"`
	OpenID      *string        `json:"openId,omitempty"`
	Sex         *int16         `json:"sex,omitempty"`
	Email       *string        `json:"email,omitempty"`
	Phone       *string        `json:"phone,omitempty"`
	AvatarURL   *string        `json:"avatarUrl,omitempty"`
	Status      int16          `json:"status"`
	LastLoginAt *time.Time     `json:"lastLoginAt,omitempty"`
	LastLoginIP *string        `json:"lastLoginIp,omitempty"`
	LoginMethod string         `json:"loginMethod"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// UserWeeklyStatisticsResponse 是当前用户本周新增资源及资源总数统计结果。
type UserWeeklyStatisticsResponse struct {
	WeeklyNewNoteCount     int64 `json:"weeklyNewNoteCount"`
	TotalNoteCount         int64 `json:"totalNoteCount"`
	WeeklyNewNotebookCount int64 `json:"weeklyNewNotebookCount"`
	TotalNotebookCount     int64 `json:"totalNotebookCount"`
	WeeklyNewFileCount     int64 `json:"weeklyNewFileCount"`
	TotalFileCount         int64 `json:"totalFileCount"`
	WeeklyNewBookmarkCount int64 `json:"weeklyNewBookmarkCount"`
	TotalBookmarkCount     int64 `json:"totalBookmarkCount"`
}

// NewUserResponse 将数据库 Model 转换为不含密码的响应 DTO。
func NewUserResponse(user model.User) UserResponse {
	return UserResponse{
		ID: user.ID, Account: user.Account, Name: user.Name, OpenID: user.OpenID,
		Roles: []RoleResponse{},
		Sex:   user.Sex, Email: user.Email, Phone: user.Phone, AvatarURL: user.AvatarURL,
		Status: user.Status, LastLoginAt: user.LastLoginAt, LastLoginIP: user.LastLoginIP,
		LoginMethod: user.LoginMethod, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

// NewUserResponseWithRoles 将用户和其关联角色转换为对外响应结构。
func NewUserResponseWithRoles(user model.User, roles []model.Role) UserResponse {
	response := NewUserResponse(user)
	response.Roles = NewRoleResponses(roles)
	return response
}

// NewUserResponses 批量转换用户响应 DTO。
func NewUserResponses(users []model.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, NewUserResponse(user))
	}
	return responses
}
