package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"note-vault/app/dto"
	"note-vault/app/model"
	"note-vault/app/repository"
	"note-vault/app/utils"
)

const (
	maxBookmarkTopicLength       = 200
	maxBookmarkLabelLength       = 100
	maxBookmarkHrefLength        = 2048
	maxBookmarkIconLength        = 1000
	maxBookmarkDescriptionLength = 2000
)

// BookmarkService 负责收藏夹业务规则、数据归属校验和 DTO/Model 转换。
type BookmarkService struct {
	bookmarks *repository.BookmarkRepository
}

// NewBookmarkService 创建收藏夹服务。
func NewBookmarkService() *BookmarkService {
	return &BookmarkService{bookmarks: repository.NewBookmarkRepository()}
}

// CreateBookmark 创建当前用户自己的收藏。
func (s *BookmarkService) CreateBookmark(ctx context.Context, ownerID uint, input dto.BookmarkRequest) (*dto.BookmarkResponse, error) {
	href, err := validateBookmarkHref(input.Href)
	if err != nil {
		return nil, err
	}
	input.Href = &href
	populateBookmarkIcon(&input, utils.GetWebsiteIconURL)
	if err := validateBookmarkFields(&input); err != nil {
		return nil, err
	}
	if input.Icon != nil && *input.Icon != "" {
		if _, err := validateBookmarkURL(input.Icon, "网站图标地址"); err != nil {
			return nil, err
		}
	}
	favorite := true
	if input.Favorite != nil {
		favorite = *input.Favorite
	}

	bookmark := model.Bookmark{
		OwnerID: ownerID, Topic: normalizeBookmarkText(input.Topic), Label: normalizeBookmarkText(input.Label),
		Href: href, Icon: normalizeBookmarkText(input.Icon), Favorite: favorite,
		Description: normalizeBookmarkText(input.Description),
	}
	if err := s.bookmarks.Create(ctx, &bookmark); err != nil {
		return nil, utils.BuildBookmarkRepositoryError(err, "创建收藏失败")
	}
	topic := ""
	if bookmark.Topic != nil {
		topic = *bookmark.Topic
	}
	auditRecorder.Record(ctx, fmt.Sprintf("新增收藏《%s》（ID %d）", topic, bookmark.ID))
	result := dto.NewBookmarkResponse(bookmark)
	return &result, nil
}

// ListBookmarks 分页查询当前用户自己的收藏，并支持多条件筛选。
func (s *BookmarkService) ListBookmarks(ctx context.Context, ownerID uint, input dto.BookmarkRequest) (*dto.PageResponse[dto.BookmarkResponse], error) {
	page, pageSize, err := utils.NormalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}
	if err := validateBookmarkTimeRange(input); err != nil {
		return nil, err
	}
	input.Page, input.PageSize = &page, &pageSize
	bookmarks, total, err := s.bookmarks.List(ctx, ownerID, input)
	if err != nil {
		return nil, utils.BuildBookmarkRepositoryError(err, "查询收藏列表失败")
	}
	return &dto.PageResponse[dto.BookmarkResponse]{
		List: dto.NewBookmarkResponses(bookmarks), Page: page, PageSize: pageSize, Total: total,
	}, nil
}

// GetBookmark 查询当前用户自己的收藏详情。
func (s *BookmarkService) GetBookmark(ctx context.Context, ownerID, id uint) (*dto.BookmarkResponse, error) {
	bookmark, err := s.bookmarks.FindByID(ctx, ownerID, id)
	if err != nil {
		return nil, utils.BuildBookmarkRepositoryError(err, "查询收藏详情失败")
	}
	result := dto.NewBookmarkResponse(*bookmark)
	return &result, nil
}

// UpdateBookmark 部分修改当前用户自己的收藏。
func (s *BookmarkService) UpdateBookmark(ctx context.Context, ownerID, id uint, input dto.BookmarkRequest) (*dto.BookmarkResponse, error) {
	if input.Topic == nil && input.Label == nil && input.Href == nil && input.Icon == nil &&
		input.Favorite == nil && input.Description == nil {
		return nil, utils.NewAppError(utils.KindInvalidArgument, "至少提供一个需要修改的收藏字段")
	}
	if _, err := s.bookmarks.FindByID(ctx, ownerID, id); err != nil {
		return nil, utils.BuildBookmarkRepositoryError(err, "查询收藏详情失败")
	}
	if input.Href != nil {
		href, err := validateBookmarkHref(input.Href)
		if err != nil {
			return nil, err
		}
		*input.Href = href
		populateBookmarkIcon(&input, utils.GetWebsiteIconURL)
	}
	if err := validateBookmarkFields(&input); err != nil {
		return nil, err
	}
	if input.Icon != nil && *input.Icon != "" {
		if _, err := validateBookmarkURL(input.Icon, "网站图标地址"); err != nil {
			return nil, err
		}
	}
	if err := s.bookmarks.Update(ctx, ownerID, id, input); err != nil {
		return nil, utils.BuildBookmarkRepositoryError(err, "更新收藏失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("修改收藏 ID %d", id))
	return s.GetBookmark(ctx, ownerID, id)
}

// DeleteBookmark 删除当前用户自己的收藏。
func (s *BookmarkService) DeleteBookmark(ctx context.Context, ownerID, id uint) error {
	if err := s.bookmarks.Delete(ctx, ownerID, id); err != nil {
		return utils.BuildBookmarkRepositoryError(err, "删除收藏失败")
	}
	auditRecorder.Record(ctx, fmt.Sprintf("删除收藏 ID %d", id))
	return nil
}

func validateBookmarkHref(value *string) (string, error) {
	href, err := utils.RequiredText(value, "收藏链接")
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(href) > maxBookmarkHrefLength {
		return "", utils.NewAppError(utils.KindInvalidArgument, "收藏链接长度不能大于 2048")
	}
	return validateBookmarkURL(&href, "收藏链接")
}

func validateBookmarkURL(value *string, fieldName string) (string, error) {
	text := strings.TrimSpace(*value)
	parsed, err := url.Parse(text)
	if err != nil || parsed.Host == "" || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return "", utils.NewAppError(utils.KindInvalidArgument, fieldName+"格式无效，仅支持 HTTP 或 HTTPS URL")
	}
	return text, nil
}

func validateBookmarkFields(input *dto.BookmarkRequest) error {
	if input.Topic != nil {
		*input.Topic = strings.TrimSpace(*input.Topic)
		if utf8.RuneCountInString(*input.Topic) > maxBookmarkTopicLength {
			return utils.NewAppError(utils.KindInvalidArgument, "收藏标题长度不能大于 200")
		}
	}
	if input.Label != nil {
		*input.Label = strings.TrimSpace(*input.Label)
		if utf8.RuneCountInString(*input.Label) > maxBookmarkLabelLength {
			return utils.NewAppError(utils.KindInvalidArgument, "收藏标签长度不能大于 100")
		}
	}
	if input.Icon != nil {
		*input.Icon = strings.TrimSpace(*input.Icon)
		if utf8.RuneCountInString(*input.Icon) > maxBookmarkIconLength {
			return utils.NewAppError(utils.KindInvalidArgument, "网站图标地址长度不能大于 1000")
		}
	}
	if input.Description != nil {
		*input.Description = strings.TrimSpace(*input.Description)
		if utf8.RuneCountInString(*input.Description) > maxBookmarkDescriptionLength {
			return utils.NewAppError(utils.KindInvalidArgument, "收藏描述长度不能大于 2000")
		}
	}
	return nil
}

func validateBookmarkTimeRange(input dto.BookmarkRequest) error {
	if input.CreatedFrom != nil && input.CreatedTo != nil && input.CreatedFrom.After(*input.CreatedTo) {
		return utils.NewAppError(utils.KindInvalidArgument, "创建时间范围无效")
	}
	return nil
}

func normalizeBookmarkText(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	return &text
}

func populateBookmarkIcon(input *dto.BookmarkRequest, fetchIcon func(string) (string, error)) {
	if input.Href == nil {
		return
	}
	iconURL, err := fetchIcon(*input.Href)
	if err != nil || utf8.RuneCountInString(iconURL) > maxBookmarkIconLength {
		iconURL = ""
	}
	input.Icon = &iconURL
}
