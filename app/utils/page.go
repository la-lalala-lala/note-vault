package utils

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePage 为分页参数设置默认值并校验取值范围。
func NormalizePage(pageInput, pageSizeInput *int) (int, int, error) {
	page := DefaultPage
	if pageInput != nil {
		page = *pageInput
	}
	pageSize := DefaultPageSize
	if pageSizeInput != nil {
		pageSize = *pageSizeInput
	}
	if page < 1 {
		return 0, 0, NewAppError(KindInvalidArgument, "分页参数 page 必须是正整数")
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		return 0, 0, NewAppError(KindInvalidArgument, "分页参数 pageSize 必须在 1 到 100 之间")
	}
	return page, pageSize, nil
}
