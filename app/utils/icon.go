package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const maxIconPageBytes = 1 << 20

// GetWebsiteIconURL 抓取网页中的图标地址；未声明图标时返回网站根目录的 favicon.ico 地址。
func GetWebsiteIconURL(rawURL string) (string, error) {
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, err
				}
				addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, address := range addresses {
					ip, ok := netip.AddrFromSlice(address.IP)
					if !ok || !publicIconIP(ip.Unmap()) {
						return nil, errors.New("网页地址不允许指向内网或保留地址")
					}
				}
				if len(addresses) == 0 {
					return nil, errors.New("网页地址没有可用的 IP")
				}
				return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(addresses[0].IP.String(), port))
			},
		},
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("网页跳转次数过多")
			}
			return validateIconPageURL(request.URL)
		},
	}
	return getWebsiteIconURL(rawURL, client)
}

func getWebsiteIconURL(rawURL string, client *http.Client) (string, error) {
	pageURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("解析网页地址失败: %w", err)
	}
	if err := validateIconPageURL(pageURL); err != nil {
		return "", err
	}
	request, err := http.NewRequest(http.MethodGet, pageURL.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("获取网页失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("获取网页失败: HTTP %d", response.StatusCode)
	}
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "html") {
		return "", errors.New("目标地址不是网页")
	}
	return extractWebsiteIconURL(io.LimitReader(response.Body, maxIconPageBytes), response.Request.URL)
}

func extractWebsiteIconURL(reader io.Reader, pageURL *url.URL) (string, error) {
	tokenizer := html.NewTokenizer(reader)
	baseURL := pageURL
	var iconPaths []string
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if err := tokenizer.Err(); err != nil && err != io.EOF {
				return "", fmt.Errorf("读取网页失败: %w", err)
			}
			break
		}
		token := tokenizer.Token()
		if tokenType == html.EndTagToken && token.Data == "head" {
			break
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		attributes := make(map[string]string, len(token.Attr))
		for _, attribute := range token.Attr {
			attributes[attribute.Key] = attribute.Val
		}
		if token.Data == "base" && attributes["href"] != "" {
			if parsed, err := url.Parse(strings.TrimSpace(attributes["href"])); err == nil {
				baseURL = pageURL.ResolveReference(parsed)
			}
		}
		if token.Data != "link" {
			continue
		}
		for _, rel := range strings.Fields(strings.ToLower(attributes["rel"])) {
			if rel == "icon" || rel == "apple-touch-icon" || rel == "apple-touch-icon-precomposed" {
				iconPaths = append(iconPaths, strings.TrimSpace(attributes["href"]))
				break
			}
		}
	}
	for _, path := range iconPaths {
		if path == "" {
			continue
		}
		iconURL, err := url.Parse(path)
		if err != nil {
			continue
		}
		iconURL = baseURL.ResolveReference(iconURL)
		if validateIconPageURL(iconURL) == nil {
			return iconURL.String(), nil
		}
	}
	return (&url.URL{Scheme: pageURL.Scheme, Host: pageURL.Host, Path: "/favicon.ico"}).String(), nil
}

func validateIconPageURL(address *url.URL) error {
	if address == nil || (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" || address.User != nil {
		return errors.New("网页地址必须是有效的 HTTP 或 HTTPS 地址")
	}
	if strings.EqualFold(address.Hostname(), "localhost") || strings.HasSuffix(strings.ToLower(address.Hostname()), ".localhost") {
		return errors.New("网页地址不允许指向本机")
	}
	if ip, err := netip.ParseAddr(address.Hostname()); err == nil && !publicIconIP(ip.Unmap()) {
		return errors.New("网页地址不允许指向内网或保留地址")
	}
	return nil
}

func publicIconIP(ip netip.Addr) bool {
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32"} {
		if netip.MustParsePrefix(prefix).Contains(ip) {
			return false
		}
	}
	return true
}
