package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServeFrontend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "入口页面", path: "/", want: "index"},
		{name: "前端路由回退", path: "/notes", want: "index"},
		{name: "静态资源", path: "/assets/app.js", want: "asset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = request

			if !serveFrontend(context, root) {
				t.Fatal("serveFrontend 返回 false")
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP 状态码 = %d，期望 %d", recorder.Code, http.StatusOK)
			}
			if recorder.Body.String() != test.want {
				t.Fatalf("响应内容 = %q，期望 %q", recorder.Body.String(), test.want)
			}
		})
	}
}

func TestServeFrontendDoesNotHandleAPI(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	if serveFrontend(context, root) {
		t.Fatal("API 路径不应回退到前端入口页面")
	}
}
