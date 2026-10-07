package dto

// PageResponse 是所有资源分页响应共用的结构。
type PageResponse[T any] struct {
	List     []T   `json:"list"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}
