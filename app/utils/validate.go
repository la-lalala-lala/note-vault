package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"note-vault/app/dto"
	"note-vault/app/model"
	"strings"
	"unicode/utf8"
)

// ValidSessionToken 校验 Token 是否为至少包含 32 字节熵的无填充 Base64URL 字符串。
func ValidSessionToken(token string) bool {
	randomBytes, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(randomBytes) >= minimumTokenBytes
}

func ValidSessionTokenHash(tokenHash string) bool {
	hashBytes, err := hex.DecodeString(tokenHash)
	return err == nil && len(hashBytes) == sha256.Size
}

func RequiredText(value *string, name string) (string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return "", NewAppError(KindInvalidArgument, name+"不能为空")
	}
	return strings.TrimSpace(*value), nil
}

func ValidUserStatus(status int16) bool {
	return status == model.UserStatusNormal || status == model.UserStatusLocked || status == model.UserStatusCancelled
}

func ValidRoleStatus(status int16) bool {
	return status == model.RoleStatusEnabled || status == model.RoleStatusDisabled
}

// ValidNotebookStatus 判断笔记簿是否处于允许由业务接口设置的正常或禁用状态。
// 删除状态只能由删除接口设置，不能通过新增或状态修改接口直接提交。
func ValidNotebookStatus(status int16) bool {
	return status == model.NotebookStatusNormal || status == model.NotebookStatusDisabled
}

func ValidNotebookOptionalFields(input *dto.NotebookRequest) error {
	if input.Icon != nil {
		icon := strings.TrimSpace(*input.Icon)
		if utf8.RuneCountInString(icon) > 100 {
			return NewAppError(KindInvalidArgument, "笔记簿封面图标名长度不能大于 100")
		}
		*input.Icon = icon
	}
	if input.Tag != nil {
		tag := strings.TrimSpace(*input.Tag)
		if utf8.RuneCountInString(tag) > 100 {
			return NewAppError(KindInvalidArgument, "笔记簿标签长度不能大于 100")
		}
		*input.Tag = tag
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if utf8.RuneCountInString(description) > 1000 {
			return NewAppError(KindInvalidArgument, "笔记簿描述长度不能大于 1000")
		}
		*input.Description = description
	}
	return nil
}

// ValidNoteOptionalFields 校验并规范化笔记的可选字段。
func ValidNoteOptionalFields(input *dto.NoteRequest) error {
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		if utf8.RuneCountInString(label) > 200 {
			return NewAppError(KindInvalidArgument, "笔记标签长度不能大于 200")
		}
		*input.Label = label
	}
	if input.Summary != nil {
		summary := strings.TrimSpace(*input.Summary)
		if utf8.RuneCountInString(summary) > 2000 {
			return NewAppError(KindInvalidArgument, "笔记摘要长度不能大于 2000")
		}
		*input.Summary = summary
	}
	if input.CoverURL != nil {
		coverURL := strings.TrimSpace(*input.CoverURL)
		if utf8.RuneCountInString(coverURL) > 500 {
			return NewAppError(KindInvalidArgument, "笔记封面地址长度不能大于 500")
		}
		*input.CoverURL = coverURL
	}
	if input.Content != nil && len(*input.Content) > 10*1024*1024 {
		return NewAppError(KindInvalidArgument, "笔记正文不能大于 10 MiB")
	}
	if input.ContentType != nil {
		contentType := strings.ToLower(strings.TrimSpace(*input.ContentType))
		if contentType != "markdown" {
			return NewAppError(KindInvalidArgument, "正文格式目前只支持 markdown")
		}
		*input.ContentType = contentType
	}
	return nil
}

// ValidDraftFields 校验并规范化草稿的可选字段。
func ValidDraftFields(input *dto.DraftRequest) error {
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if utf8.RuneCountInString(title) > 200 {
			return NewAppError(KindInvalidArgument, "草稿标题长度不能大于 200")
		}
		*input.Title = title
	}
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		if utf8.RuneCountInString(label) > 200 {
			return NewAppError(KindInvalidArgument, "草稿标签长度不能大于 200")
		}
		*input.Label = label
	}
	if input.Content != nil && len(*input.Content) > 10*1024*1024 {
		return NewAppError(KindInvalidArgument, "草稿正文不能大于 10 MiB")
	}
	if input.ContentType != nil {
		contentType := strings.ToLower(strings.TrimSpace(*input.ContentType))
		if contentType != "markdown" {
			return NewAppError(KindInvalidArgument, "正文格式目前只支持 markdown")
		}
		*input.ContentType = contentType
	}
	return nil
}
