package service

import (
	"errors"
	"strings"
	"testing"

	"note-vault/app/dto"
)

func TestPopulateBookmarkIcon(t *testing.T) {
	const href = "https://zhuanlan.zhihu.com/p/27700617"
	t.Run("创建或修改链接时自动覆盖客户端图标", func(t *testing.T) {
		manualIcon := "https://example.com/manual.ico"
		input := dto.BookmarkRequest{Href: stringPointer(href), Icon: &manualIcon}
		populateBookmarkIcon(&input, func(receivedHref string) (string, error) {
			if receivedHref != href {
				t.Fatalf("抓取地址 = %q，期望 %q", receivedHref, href)
			}
			return "https://static.zhihu.com/favicon.ico", nil
		})
		if input.Icon == nil || *input.Icon != "https://static.zhihu.com/favicon.ico" {
			t.Errorf("自动生成的图标 = %v", input.Icon)
		}
	})

	t.Run("抓取失败时图标留空", func(t *testing.T) {
		oldIcon := "https://example.com/old.ico"
		input := dto.BookmarkRequest{Href: stringPointer(href), Icon: &oldIcon}
		populateBookmarkIcon(&input, func(string) (string, error) {
			return "", errors.New("网站暂不可访问")
		})
		if input.Icon == nil || *input.Icon != "" {
			t.Errorf("抓取失败的图标 = %v，期望空字符串", input.Icon)
		}
	})

	t.Run("修改其他字段时不抓取或覆盖图标", func(t *testing.T) {
		manualIcon := "https://example.com/manual.ico"
		input := dto.BookmarkRequest{Icon: &manualIcon}
		populateBookmarkIcon(&input, func(string) (string, error) {
			t.Fatal("未更新 href 时不应抓取")
			return "", nil
		})
		if input.Icon == nil || *input.Icon != manualIcon {
			t.Errorf("原图标 = %v，期望 %q", input.Icon, manualIcon)
		}
	})

	t.Run("过长图标不写入", func(t *testing.T) {
		input := dto.BookmarkRequest{Href: stringPointer(href)}
		populateBookmarkIcon(&input, func(string) (string, error) {
			return "https://example.com/" + strings.Repeat("a", maxBookmarkIconLength), nil
		})
		if input.Icon == nil || *input.Icon != "" {
			t.Errorf("过长图标 = %v，期望空字符串", input.Icon)
		}
	})
}

func stringPointer(value string) *string {
	return &value
}
