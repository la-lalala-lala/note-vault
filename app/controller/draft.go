package controller

import (
	"net/http"

	"note-vault/app/dto"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var draftService = service.NewDraftService()

func SaveDraft(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.DraftRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := draftService.SaveDraft(c.Request.Context(), user.UserID, c.Param("draftKey"), input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "保存草稿成功", result)
}

func ListDrafts(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.DraftRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := draftService.ListDrafts(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func GetDraft(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "草稿 ID 必须是正整数")
		return
	}
	result, err := draftService.GetDraft(c.Request.Context(), user.UserID, id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func SubmitDraft(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "草稿 ID 必须是正整数")
		return
	}
	var input dto.DraftRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := draftService.SubmitDraft(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "提交草稿成功", result)
}

func DeleteDraft(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "草稿 ID 必须是正整数")
		return
	}
	if err := draftService.DeleteDraft(c.Request.Context(), user.UserID, id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除草稿成功", nil)
}
