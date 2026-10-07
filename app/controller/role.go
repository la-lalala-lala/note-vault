package controller

import (
	"net/http"
	"note-vault/app/utils"

	"note-vault/app/dto"
	"note-vault/app/service"

	"github.com/gin-gonic/gin"
)

var roleService = service.NewRoleService()

func ListRoles(c *gin.Context) {
	var input dto.RoleRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.ListRoles(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func CreateRole(c *gin.Context) {
	var input dto.RoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.CreateRole(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "创建角色成功", result)
}

func GetRole(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色 ID 必须是正整数")
		return
	}
	result, err := roleService.GetRole(c.Request.Context(), id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func UpdateRole(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色 ID 必须是正整数")
		return
	}
	var input dto.RoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.UpdateRole(c.Request.Context(), id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新角色成功", result)
}

func UpdateRoleStatus(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色 ID 必须是正整数")
		return
	}
	var input dto.RoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if input.Status == nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色状态不能为空")
		return
	}
	result, err := roleService.UpdateRoleStatus(c.Request.Context(), id, *input.Status)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "角色状态更新成功", result)
}

func DeleteRole(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色 ID 必须是正整数")
		return
	}
	if err := roleService.DeleteRole(c.Request.Context(), id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除角色成功", nil)
}

func AssignUserRoles(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "用户 ID 必须是正整数")
		return
	}
	var input dto.IDsRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if err := roleService.AssignUserRoles(c.Request.Context(), id, input); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "用户角色更新成功", nil)
}

func AssignRoleMenus(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "角色 ID 必须是正整数")
		return
	}
	var input dto.IDsRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if err := roleService.AssignRoleMenus(c.Request.Context(), id, input); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "角色菜单权限更新成功", nil)
}

// 返回菜单树
func ListMenuTree(c *gin.Context) {
	var input dto.MenuRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.ListMenuTree(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func CreateMenu(c *gin.Context) {
	var input dto.MenuRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.CreateMenu(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "创建菜单成功", result)
}

func UpdateMenu(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "菜单 ID 必须是正整数")
		return
	}
	var input dto.MenuRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := roleService.UpdateMenu(c.Request.Context(), id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新菜单成功", result)
}

func DeleteMenu(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "菜单 ID 必须是正整数")
		return
	}
	if err := roleService.DeleteMenu(c.Request.Context(), id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除菜单成功", nil)
}
