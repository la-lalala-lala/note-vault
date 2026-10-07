package dto

import (
	"time"

	"note-vault/app/model"
)

// RoleRequest 是角色新增、修改和查询共用的请求结构。
type RoleRequest struct {
	Name        *string `json:"name" form:"name" binding:"omitempty,max=100"`
	Code        *string `json:"code" form:"code" binding:"omitempty,max=100"`
	Description *string `json:"description" form:"description" binding:"omitempty,max=500"`
	Status      *int16  `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
	Page        *int    `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize    *int    `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=100"`
}

// MenuRequest 是菜单新增、修改和树查询共用的请求结构。
type MenuRequest struct {
	ParentID   *uint   `json:"parentId" form:"parentId"`
	Name       *string `json:"name" form:"name" binding:"omitempty,max=100"`
	Code       *string `json:"code" form:"code" binding:"omitempty,max=150"`
	Type       *int16  `json:"type" form:"type" binding:"omitempty,oneof=1 2 3 4"`
	RoutePath  *string `json:"routePath" form:"routePath" binding:"omitempty,max=300"`
	Component  *string `json:"component" form:"component" binding:"omitempty,max=300"`
	HTTPMethod *string `json:"httpMethod" form:"httpMethod" binding:"omitempty,oneof=GET POST PUT PATCH DELETE OPTIONS"`
	APIPath    *string `json:"apiPath" form:"apiPath" binding:"omitempty,max=500"`
	SortOrder  *int    `json:"sortOrder" form:"sortOrder"`
	Visible    *bool   `json:"visible" form:"visible"`
	Status     *int16  `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

// IDsRequest 是用户角色和角色菜单授权共用的 ID 列表请求。
type IDsRequest struct {
	IDs []uint `json:"ids" binding:"omitempty,dive,min=1"`
}

// RoleResponse 是角色对外返回结构。
type RoleResponse struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	Code        string          `json:"code"`
	Description *string         `json:"description,omitempty"`
	Status      int16           `json:"status"`
	Menus       *[]MenuResponse `json:"menus,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// MenuResponse 是菜单和权限资源对外返回结构，Children 用于树形结果。
type MenuResponse struct {
	ID         uint           `json:"id"`
	ParentID   *uint          `json:"parentId,omitempty"`
	Name       string         `json:"name"`
	Code       string         `json:"code"`
	Type       int16          `json:"type"`
	RoutePath  *string        `json:"routePath,omitempty"`
	Component  *string        `json:"component,omitempty"`
	HTTPMethod *string        `json:"httpMethod,omitempty"`
	APIPath    *string        `json:"apiPath,omitempty"`
	SortOrder  int            `json:"sortOrder"`
	Visible    bool           `json:"visible"`
	Status     int16          `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	Children   []MenuResponse `json:"children,omitempty"`
}

func NewRoleResponse(role model.Role) RoleResponse {
	return RoleResponse{
		ID: role.ID, Name: role.Name, Code: role.Code, Description: role.Description,
		Status: role.Status, CreatedAt: role.CreatedAt, UpdatedAt: role.UpdatedAt,
	}
}

// NewRoleResponseWithMenus 将角色及其关联的菜单权限资源转换为详情响应。
func NewRoleResponseWithMenus(role model.Role, menus []model.Menu) RoleResponse {
	response := NewRoleResponse(role)
	menuResponses := NewMenuResponses(menus)
	response.Menus = &menuResponses
	return response
}

func NewRoleResponses(roles []model.Role) []RoleResponse {
	responses := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		responses = append(responses, NewRoleResponse(role))
	}
	return responses
}

func NewMenuResponses(menus []model.Menu) []MenuResponse {
	responses := make([]MenuResponse, 0, len(menus))
	for _, menu := range menus {
		responses = append(responses, NewMenuResponse(menu))
	}
	return responses
}

func NewMenuResponse(menu model.Menu) MenuResponse {
	return MenuResponse{
		ID: menu.ID, ParentID: menu.ParentID, Name: menu.Name, Code: menu.Code,
		Type: menu.Type, RoutePath: menu.RoutePath, Component: menu.Component,
		HTTPMethod: menu.HTTPMethod, APIPath: menu.APIPath, SortOrder: menu.SortOrder,
		Visible: menu.Visible, Status: menu.Status, CreatedAt: menu.CreatedAt,
		UpdatedAt: menu.UpdatedAt,
	}
}
