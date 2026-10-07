package utils

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrOptimisticLockConflict 表示更新条件中的版本号已经过期。
var ErrOptimisticLockConflict = errors.New("optimistic lock conflict")

// ErrDraftAssociationConflict 表示草稿不能切换到另一条正式笔记。
var ErrDraftAssociationConflict = errors.New("draft association conflict")

// ErrNotebookUnavailable 表示草稿提交时目标笔记簿已禁用或已删除。
var ErrNotebookUnavailable = errors.New("notebook is unavailable")

// ErrTargetNotebookRequired 表示提交草稿时没有可用的目标笔记簿。
var ErrTargetNotebookRequired = errors.New("target notebook is required")

// ErrNoteVersionRequired 表示编辑已有笔记的草稿提交请求缺少正式笔记版本号。
var ErrNoteVersionRequired = errors.New("note version is required")

// ErrNoteAlreadyTrashed 表示试图重复将回收站中的笔记移入回收站。
var ErrNoteAlreadyTrashed = errors.New("note is already in trash")

// ErrNoteNotTrashed 表示恢复或永久删除的目标不是回收站笔记。
var ErrNoteNotTrashed = errors.New("note is not in trash")

// Kind 标识可由 HTTP 层稳定转换的应用错误类别。
type Kind uint8

const (
	KindInvalidArgument Kind = iota + 1
	KindUnauthenticated
	KindPermissionDenied
	KindNotFound
	KindConflict
	KindVersionConflict
	KindDraftAssociationConflict
	KindUnavailable
	KindInternal
)

// AppError 是 Service 返回给上层的业务错误，不携带 HTTP 实现细节。
type AppError struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 暴露底层错误供 errors.Is / errors.As 使用。
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError 创建不包装底层错误的应用错误。
func NewAppError(kind Kind, message string) *AppError {
	return &AppError{Kind: kind, Message: message}
}

// WrapAppError 创建包装底层错误的应用错误。
func WrapAppError(kind Kind, message string, cause error) *AppError {
	return &AppError{Kind: kind, Message: message, Cause: cause}
}

func BuildRoleRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "角色或菜单不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "编码已存在或关联关系重复")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

func BuildUserRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "用户不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "用户账号已存在")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

func BuildNotebookRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "笔记簿不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "同一用户下笔记簿名已存在")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

func BuildNoteRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, ErrNoteAlreadyTrashed):
		return NewAppError(KindConflict, "笔记已经在回收站中")
	case errors.Is(err, ErrNoteNotTrashed):
		return NewAppError(KindConflict, "只能操作回收站中的笔记")
	case errors.Is(err, ErrOptimisticLockConflict):
		return NewAppError(KindVersionConflict, "笔记已被其他操作更新，请重新加载后再保存")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "笔记不存在")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

func BuildDraftRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, ErrTargetNotebookRequired):
		return NewAppError(KindInvalidArgument, "提交草稿必须指定目标笔记簿")
	case errors.Is(err, ErrNoteVersionRequired):
		return NewAppError(KindInvalidArgument, "提交已有笔记的草稿必须提交正式笔记版本号")
	case errors.Is(err, ErrOptimisticLockConflict):
		return NewAppError(KindVersionConflict, "正式笔记已被其他操作更新，草稿已保留，请重新加载或合并")
	case errors.Is(err, ErrDraftAssociationConflict):
		return NewAppError(KindDraftAssociationConflict, "草稿关联信息已变化，请重新加载草稿")
	case errors.Is(err, ErrNotebookUnavailable):
		return NewAppError(KindConflict, "目标笔记簿当前不可用")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "草稿或关联资源不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "该笔记已有未提交草稿")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

// BuildFileRepositoryError 将文件元数据仓储错误转换为统一业务错误。
func BuildFileRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "文件不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "文件对象键已存在")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}

// BuildBookmarkRepositoryError 将收藏夹仓储错误转换为统一业务错误。
func BuildBookmarkRepositoryError(err error, message string) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return NewAppError(KindNotFound, "收藏不存在")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return NewAppError(KindConflict, "同一用户下该 URL 已收藏")
	default:
		return WrapAppError(KindInternal, message, err)
	}
}
