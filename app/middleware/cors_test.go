package middleware

import (
	"strings"
	"testing"
)

func TestContentSecurityPolicy(t *testing.T) {
	// 接口响应必须保持严格 CSP，避免误放开 API 响应的资源加载能力。
	apiPolicy := contentSecurityPolicy("/api/v1/auth/login")
	if apiPolicy != "default-src 'none'; frame-ancestors 'none'" {
		t.Fatalf("API CSP = %q", apiPolicy)
	}

	// 前端页面必须允许加载脚本、样式，并访问同源后端接口。
	frontendPolicy := contentSecurityPolicy("/notes")
	for _, directive := range []string{"default-src 'self'", "script-src 'self'", "style-src 'self' 'unsafe-inline'", "connect-src 'self'"} {
		if !containsPolicyDirective(frontendPolicy, directive) {
			t.Fatalf("前端 CSP 缺少 %q：%s", directive, frontendPolicy)
		}
	}
}

func containsPolicyDirective(policy, directive string) bool {
	// 使用字符串包含判断即可验证目标指令是否出现在 CSP 中。
	return strings.Contains(policy, directive)
}
