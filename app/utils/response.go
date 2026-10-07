package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	// HeaderRequestID 是请求 ID 使用的标准响应头名称。
	HeaderRequestID = "X-Request-ID"

	// ContextKeyRequestID 是 Gin Context 中保存请求 ID 的键。
	ContextKeyRequestID = "requestID"
)

// Code 是客户端可稳定识别的业务响应码。
type Code int

const (
	CodeSuccess            Code = 0
	CodeInvalidArgument    Code = 40001
	CodeUnauthenticated    Code = 40101
	CodePermissionDenied   Code = 40301
	CodeNotFound           Code = 40401
	CodeConflict           Code = 40901
	CodeVersionConflict    Code = 40902
	CodePayloadTooLarge    Code = 41301
	CodeInternalError      Code = 50001
	CodeServiceUnavailable Code = 50301
)

// Body 是所有 JSON 接口统一使用的响应结构。
type Body struct {
	Code      Code   `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"requestId"`
}

// HTTPError 保存一次已经转换为 HTTP 响应的错误。
// 它会写入 Gin 错误集合，由错误日志中间件统一记录，避免错误返回给客户端后丢失排查信息。
type HTTPError struct {
	HTTPStatus int
	Code       Code
	Message    string
	Cause      error
}

func (e *HTTPError) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

// Unwrap 暴露原始错误，便于日志层使用 errors.As/errors.Is 继续识别错误类型。
func (e *HTTPError) Unwrap() error {
	return e.Cause
}

// PageData 是分页接口统一使用的数据结构。
type PageData struct {
	List     any   `json:"list"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// Success 返回统一成功响应。
func Success(c *gin.Context, data any) {
	SuccessWithMessage(c, "success", data)
}

// SuccessWithMessage 返回带业务提示的统一成功响应。
func SuccessWithMessage(c *gin.Context, message string, data any) {
	write(c, http.StatusOK, CodeSuccess, message, data)
}

// Page 返回统一分页响应。
func Page(c *gin.Context, list any, page, pageSize int, total int64) {
	Success(c, PageData{
		List:     list,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Error 返回统一错误响应。
func Error(c *gin.Context, httpStatus int, code Code, message string) {
	writeError(c, httpStatus, code, message, nil, nil)
}

// ErrorWithData 返回携带安全诊断数据的统一错误响应。
func ErrorWithData(c *gin.Context, httpStatus int, code Code, message string, data any) {
	writeError(c, httpStatus, code, message, data, nil)
}

// Abort 返回统一错误响应并终止 Gin 后续处理链。
func Abort(c *gin.Context, httpStatus int, code Code, message string) {
	Error(c, httpStatus, code, message)
	c.Abort()
}

// ErrorFrom 将应用错误转换为统一 HTTP 错误响应。
func ErrorFrom(c *gin.Context, err error) {
	var applicationError *AppError
	if !errors.As(err, &applicationError) {
		writeError(c, http.StatusInternalServerError, CodeInternalError, "系统内部错误", nil, err)
		return
	}

	switch applicationError.Kind {
	case KindInvalidArgument:
		writeError(c, http.StatusBadRequest, CodeInvalidArgument, applicationError.Message, nil, err)
	case KindUnauthenticated:
		writeError(c, http.StatusUnauthorized, CodeUnauthenticated, applicationError.Message, nil, err)
	case KindPermissionDenied:
		writeError(c, http.StatusForbidden, CodePermissionDenied, applicationError.Message, nil, err)
	case KindNotFound:
		writeError(c, http.StatusNotFound, CodeNotFound, applicationError.Message, nil, err)
	case KindConflict:
		writeError(c, http.StatusConflict, CodeConflict, applicationError.Message, nil, err)
	case KindVersionConflict:
		writeError(c, http.StatusConflict, CodeVersionConflict, applicationError.Message, nil, err)
	case KindDraftAssociationConflict:
		writeError(c, http.StatusConflict, CodeConflict, applicationError.Message, nil, err)
	case KindUnavailable:
		writeError(c, http.StatusServiceUnavailable, CodeServiceUnavailable, applicationError.Message, nil, err)
	default:
		writeError(c, http.StatusInternalServerError, CodeInternalError, "系统内部错误", nil, err)
	}
}

// ValidationError 将 JSON 解析、字段校验和请求体大小错误转换为统一参数错误。
func ValidationError(c *gin.Context, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		writeError(c, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "请求体大小超过限制", nil, err)
		return
	}

	writeError(c, http.StatusBadRequest, CodeInvalidArgument, validationMessage(err), nil, err)
}

func write(c *gin.Context, httpStatus int, code Code, message string, data any) {
	c.JSON(httpStatus, Body{
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: c.GetString(ContextKeyRequestID),
	})
}

func writeError(c *gin.Context, httpStatus int, code Code, message string, data any, cause error) {
	c.Error(&HTTPError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Cause:      cause,
	})
	write(c, httpStatus, code, message, data)
}

func validationMessage(err error) string {
	if errors.Is(err, io.EOF) {
		return "请求体不能为空"
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return "请求体不是有效的 JSON"
	}

	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return fmt.Sprintf("字段 %s 类型错误", typeError.Field)
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) && len(validationErrors) > 0 {
		fieldError := validationErrors[0]
		switch fieldError.Tag() {
		case "required":
			return fmt.Sprintf("字段 %s 不能为空", fieldError.Field())
		case "email":
			return fmt.Sprintf("字段 %s 必须是有效邮箱", fieldError.Field())
		case "min":
			return fmt.Sprintf("字段 %s 长度不能小于 %s", fieldError.Field(), fieldError.Param())
		case "max":
			return fmt.Sprintf("字段 %s 长度不能大于 %s", fieldError.Field(), fieldError.Param())
		default:
			return fmt.Sprintf("字段 %s 校验失败", fieldError.Field())
		}
	}

	return "请求参数格式错误"
}
