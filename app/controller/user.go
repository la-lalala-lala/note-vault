package controller

import (
	"errors"
	"net/http"
	"note-vault/app/utils"
	"strconv"

	"note-vault/app/dto"
	"note-vault/app/service"

	"github.com/gin-gonic/gin"
)

var userService = service.NewUserService()

// Register 注册用户（暂时不提供对外服务）。
func Register(c *gin.Context) {
	var input dto.UserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	user, err := userService.Register(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "注册成功", user)
}

// CreateUser 由管理员新增用户，使用独立的 POST:/api/v1/users 权限。
func CreateUser(c *gin.Context) {
	var input dto.UserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	user, err := userService.CreateUser(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "新增用户成功", user)
}

// ListUsers 获取用户分页列表。
func ListUsers(c *gin.Context) {
	var input dto.UserRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	result, err := userService.ListUsers(c.Request.Context(), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// GetUser 获取用户详情。
func GetUser(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "用户 ID 必须是正整数")
		return
	}

	user, err := userService.GetUser(c.Request.Context(), id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, user)
}

// GetCurrentUser 获取当前登录用户资料。
func GetCurrentUser(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}

	result, err := userService.GetUser(c.Request.Context(), user.UserID)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// GetCurrentUserWeeklyStatistics 获取当前用户本周新增资源统计。
func GetCurrentUserWeeklyStatistics(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	result, err := userService.GetCurrentUserWeeklyStatistics(c.Request.Context(), user.UserID)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// UpdateUser 更新用户信息。
func UpdateUser(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "用户 ID 必须是正整数")
		return
	}

	var input dto.UserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	user, err := userService.UpdateUser(c.Request.Context(), id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新成功", user)
}

// UpdateCurrentUser 只使用会话中的用户 ID 更新本人资料，客户端不能指定其他用户 ID。
func UpdateCurrentUser(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}

	var input dto.UserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	result, err := userService.UpdateUser(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新成功", result)
}

// UpdateUserStatus 由管理员切换用户正常、锁定和注销状态。
func UpdateUserStatus(c *gin.Context) {
	_, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "用户 ID 必须是正整数")
		return
	}

	var input dto.UserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}

	result, err := userService.UpdateUserStatus(c.Request.Context(), id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "用户状态更新成功", result)
}

// CancelUser 注销用户，不物理删除用户记录。
func CancelUser(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}

	result, err := userService.CancelUser(c.Request.Context(), user)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "用户已注销", result)
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if err := userService.ChangePassword(c.Request.Context(), user.UserID, input); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "密码修改成功，请重新登录", nil)
}

func parseID(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid ID")
	}
	return uint(id), nil
}
