package controller

import (
	"net/http"
	"note-vault/app/dto"
	"note-vault/app/middleware"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var authService = service.NewAuthService()

func Login(c *gin.Context) {
	var input dto.LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := authService.Login(c.Request.Context(), input, c.ClientIP())
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "登录成功", result)
}

func Logout(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	tokenHash := c.GetString(middleware.ContextKeySessionTokenHash)
	if tokenHash == "" {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前会话无效")
		return
	}
	if err := authService.Logout(c.Request.Context(), tokenHash, user.Account); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "退出登录成功", nil)
}
