package controller

import (
	"net/http"

	"note-vault/app/dto"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var notebookService = service.NewNotebookService()

func CreateNotebook(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}

	var input dto.NotebookRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := notebookService.CreateNotebook(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "创建笔记簿成功", result)
}

func ListNotebooks(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.NotebookRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := notebookService.ListNotebooks(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func GetNotebook(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记簿 ID 必须是正整数")
		return
	}
	result, err := notebookService.GetNotebook(c.Request.Context(), user.UserID, id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func UpdateNotebook(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记簿 ID 必须是正整数")
		return
	}
	var input dto.NotebookRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := notebookService.UpdateNotebook(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新笔记簿成功", result)
}

func UpdateNotebookSort(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记簿 ID 必须是正整数")
		return
	}
	var input dto.NotebookRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := notebookService.UpdateNotebookSort(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新笔记簿排序成功", result)
}

func UpdateNotebookStatus(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记簿 ID 必须是正整数")
		return
	}
	var input dto.NotebookRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := notebookService.UpdateNotebookStatus(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "更新笔记簿状态成功", result)
}

func DeleteNotebook(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记簿 ID 必须是正整数")
		return
	}
	if err := notebookService.DeleteNotebook(c.Request.Context(), user.UserID, id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除笔记簿成功", nil)
}
