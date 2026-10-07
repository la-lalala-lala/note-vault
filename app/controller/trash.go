package controller

import (
	"errors"
	"io"
	"net/http"

	"note-vault/app/dto"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

func ListTrashNotes(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.NoteRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := noteService.ListTrashNotes(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

func RestoreTrashNote(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记 ID 必须是正整数")
		return
	}
	var input dto.NoteRequest
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		utils.ValidationError(c, err)
		return
	}
	result, err := noteService.RestoreNote(c.Request.Context(), user.UserID, id, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "恢复笔记成功", result)
}

func PermanentlyDeleteTrashNote(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "笔记 ID 必须是正整数")
		return
	}
	if err := noteService.PermanentlyDeleteNote(c.Request.Context(), user.UserID, id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "永久删除笔记成功", nil)
}
