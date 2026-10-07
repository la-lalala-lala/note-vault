package controller

import (
	"mime"
	"net/http"
	"strings"

	"note-vault/app/dto"
	"note-vault/app/service"
	"note-vault/app/utils"

	"github.com/gin-gonic/gin"
)

var fileService = service.NewFileService()

// UploadFile 接收 multipart/form-data 中名为 file 的文件并保存文件元数据。
func UploadFile(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "上传文件不能为空")
		return
	}
	result, err := fileService.UploadFile(c.Request.Context(), user.UserID, fileHeader)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "上传文件成功", result)
}

// UploadBase64Image 接收 JSON 格式的 Base64 图片并保存文件元数据。
func UploadBase64Image(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.Base64ImageRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := fileService.UploadBase64Image(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "上传 Base64 图片成功", result)
}

// ListFiles 分页查询当前用户自己的文件。
func ListFiles(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	var input dto.FileRequest
	if err := c.ShouldBindQuery(&input); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := fileService.ListFiles(c.Request.Context(), user.UserID, input)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// GetFile 查询当前用户自己的文件详情。
func GetFile(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "文件 ID 必须是正整数")
		return
	}
	result, err := fileService.GetFile(c.Request.Context(), user.UserID, id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.Success(c, result)
}

// PreviewFile 返回正常文件内容，用于无需登录的图片、PDF 等文件预览。
func PreviewFile(c *gin.Context) {
	publicKey := strings.TrimSpace(c.Param("publicKey"))
	if publicKey == "" {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "文件公开访问标识不能为空")
		return
	}
	file, reader, err := fileService.OpenPublicFile(c.Request.Context(), publicKey)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if file.FileType != nil && *file.FileType != "" {
		contentType = *file.FileType
	}
	contentDisposition := mime.FormatMediaType("inline", map[string]string{"filename": file.FileName})
	c.Header("Content-Disposition", contentDisposition)
	c.DataFromReader(http.StatusOK, file.FileSize, contentType, reader, nil)
}

// DownloadFile 返回当前用户自己的文件内容，并提示客户端下载保存。
func DownloadFile(c *gin.Context) {
	streamFile(c, "attachment")
}

func streamFile(c *gin.Context, disposition string) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "文件 ID 必须是正整数")
		return
	}
	file, reader, err := fileService.OpenFile(c.Request.Context(), user.UserID, id)
	if err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if file.FileType != nil && *file.FileType != "" {
		contentType = *file.FileType
	}
	contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": file.FileName})
	c.Header("Content-Disposition", contentDisposition)
	c.DataFromReader(http.StatusOK, file.FileSize, contentType, reader, nil)
}

// DeleteFile 删除当前用户自己的文件对象，并将元数据标记为已删除。
func DeleteFile(c *gin.Context) {
	user, ok := utils.CurrentSession(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, utils.CodeUnauthenticated, "当前用户身份无效")
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, utils.CodeInvalidArgument, "文件 ID 必须是正整数")
		return
	}
	if err := fileService.DeleteFile(c.Request.Context(), user.UserID, id); err != nil {
		utils.ErrorFrom(c, err)
		return
	}
	utils.SuccessWithMessage(c, "删除文件成功", nil)
}
