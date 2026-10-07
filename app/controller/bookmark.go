package controller

import (
	"net/http"

	"note-vault/app/dto"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var bookmarkService = service.NewBookmarkService()

// CreateBookmark 创建当前用户自己的收藏。
func CreateBookmark(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.BookmarkRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := bookmarkService.CreateBookmark(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "创建收藏成功", result)
}

// ListBookmarks 分页查询当前用户自己的收藏。
func ListBookmarks(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.BookmarkRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	if c.Query("favorite") == "" {
		input.Favorite = nil
	}
	if c.Query("createdFrom") == "" {
		input.CreatedFrom = nil
	}
	if c.Query("createdTo") == "" {
		input.CreatedTo = nil
	}
	result, err := bookmarkService.ListBookmarks(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// GetBookmark 查询当前用户自己的收藏详情。
func GetBookmark(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "收藏 ID 必须是正整数")
		return
	}
	result, err := bookmarkService.GetBookmark(c.Request.Context(), user.UserID, id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// UpdateBookmark 部分修改当前用户自己的收藏。
func UpdateBookmark(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "收藏 ID 必须是正整数")
		return
	}
	var input dto.BookmarkRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := bookmarkService.UpdateBookmark(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新收藏成功", result)
}

// DeleteBookmark 删除当前用户自己的收藏。
func DeleteBookmark(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "收藏 ID 必须是正整数")
		return
	}
	if err := bookmarkService.DeleteBookmark(c.Request.Context(), user.UserID, id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除收藏成功", nil)
}
