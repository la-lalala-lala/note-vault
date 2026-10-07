package router

import (
	"fmt"
	"net/http"
	"note-vault/app/utils"
	"os"
	"path"
	"path/filepath"
	"strings"

	"note-vault/app/config"
	"note-vault/app/controller"
	"note-vault/app/middleware"

	"github.com/gin-gonic/gin"
)

func InitRouter() (*gin.Engine, error) {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(config.AppConfig.Auth.TrustedProxies); err != nil {
		return nil, fmt.Errorf("配置 Gin 可信代理失败: %w", err)
	}

	r.Use(
		middleware.RequestID(),
		middleware.ErrorLogger(),
		middleware.Recovery(),
		middleware.SecurityHeaders(),
		middleware.CORS(config.AppConfig.CORS),
		middleware.BodyLimit(config.AppConfig.Server.MaxBodyBytes),
	)

	frontendRoot := config.AppConfig.Web.DistDir
	r.NoRoute(func(c *gin.Context) {
		if serveFrontend(c, frontendRoot) {
			return
		}
		utils.Error(c, http.StatusNotFound, utils.CodeNotFound, "接口不存在")
	})
	r.NoMethod(func(c *gin.Context) {
		utils.Error(c, http.StatusMethodNotAllowed, utils.CodeInvalidArgument, "请求方法不被允许")
	})

	health := r.Group("/health")
	{
		health.GET("/live", controller.Live)
		health.GET("/ready", controller.Ready)
	}

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", controller.Login)
		api.GET("/files/public/:publicKey/preview", controller.PreviewFile)

		protected := api.Group("")
		protected.Use(middleware.SessionAuthMiddleware())
		{
			protected.GET("/audit-logs", controller.ListAuditLogs)
			protected.GET("/auth/logout", controller.Logout)

			protected.PUT("/users/password", controller.ChangePassword)
			protected.GET("/users/profile", controller.GetCurrentUser)
			protected.PUT("/users/profile", controller.UpdateCurrentUser)
			protected.GET("/users/statistics/weekly", controller.GetCurrentUserWeeklyStatistics)
			protected.POST("/users", controller.CreateUser)
			protected.GET("/users", controller.ListUsers)
			protected.GET("/users/:id", controller.GetUser)
			protected.PUT("/users/:id", controller.UpdateUser)
			protected.PUT("/users/:id/status", controller.UpdateUserStatus)
			protected.DELETE("/users/:id", controller.CancelUser)
			protected.PUT("/users/:id/roles", controller.AssignUserRoles)

			protected.GET("/roles", controller.ListRoles)
			protected.POST("/roles", controller.CreateRole)
			protected.GET("/roles/:id", controller.GetRole)
			protected.PUT("/roles/:id", controller.UpdateRole)
			protected.PUT("/roles/:id/status", controller.UpdateRoleStatus)
			protected.DELETE("/roles/:id", controller.DeleteRole)
			protected.PUT("/roles/:id/menus", controller.AssignRoleMenus)

			protected.GET("/menus/tree", controller.ListMenuTree)
			protected.POST("/menus", controller.CreateMenu)
			protected.PUT("/menus/:id", controller.UpdateMenu)
			protected.DELETE("/menus/:id", controller.DeleteMenu)

			protected.GET("/notebooks", controller.ListNotebooks)
			protected.POST("/notebooks", controller.CreateNotebook)
			protected.GET("/notebooks/:id", controller.GetNotebook)
			protected.PUT("/notebooks/:id", controller.UpdateNotebook)
			protected.PUT("/notebooks/:id/sort", controller.UpdateNotebookSort)
			protected.PUT("/notebooks/:id/status", controller.UpdateNotebookStatus)
			protected.DELETE("/notebooks/:id", controller.DeleteNotebook)

			protected.GET("/notes", controller.ListNotes)
			protected.POST("/notes", controller.CreateNote)
			protected.GET("/notes/:id", controller.GetNote)
			protected.PUT("/notes/:id", controller.UpdateNote)
			protected.PUT("/notes/:id/move", controller.MoveNote)
			protected.DELETE("/notes/:id", controller.DeleteNote)
			protected.PUT("/notes/:id/restore", controller.RestoreTrashNote)
			protected.DELETE("/notes/:id/permanent", controller.PermanentlyDeleteTrashNote)

			protected.GET("/trash/notes", controller.ListTrashNotes)
			protected.PUT("/trash/notes/:id/restore", controller.RestoreTrashNote)
			protected.DELETE("/trash/notes/:id", controller.PermanentlyDeleteTrashNote)

			protected.GET("/bookmarks", controller.ListBookmarks)
			protected.POST("/bookmarks", controller.CreateBookmark)
			protected.GET("/bookmarks/:id", controller.GetBookmark)
			protected.PUT("/bookmarks/:id", controller.UpdateBookmark)
			protected.DELETE("/bookmarks/:id", controller.DeleteBookmark)

			protected.GET("/drafts", controller.ListDrafts)
			protected.PUT("/drafts/:draftKey", controller.SaveDraft)
			protected.GET("/drafts/:id", controller.GetDraft)
			protected.POST("/drafts/:id/submit", controller.SubmitDraft)
			protected.DELETE("/drafts/:id", controller.DeleteDraft)

			protected.POST("/files", controller.UploadFile)
			protected.POST("/files/base64", controller.UploadBase64Image)
			protected.GET("/files", controller.ListFiles)
			protected.GET("/files/:id", controller.GetFile)
			protected.GET("/files/:id/download", controller.DownloadFile)
			protected.DELETE("/files/:id", controller.DeleteFile)
		}
	}

	return r, nil
}

// serveFrontend 为前端静态资源提供服务，并将前端路由回退到 index.html。
func serveFrontend(c *gin.Context, root string) bool {
	if strings.TrimSpace(root) == "" || (c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead) {
		return false
	}

	requestPath := path.Clean(c.Request.URL.Path)
	if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/health" || strings.HasPrefix(requestPath, "/health/") {
		return false
	}

	// 带扩展名的请求通常是 JS、CSS、图片或字体，直接交给静态文件服务器。
	if filepath.Ext(requestPath) != "" {
		http.FileServer(http.Dir(root)).ServeHTTP(c.Writer, c.Request)
		return true
	}

	// React Router 的页面路径没有真实文件，需要统一返回入口页面。
	indexPath := filepath.Join(root, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	http.ServeFile(c.Writer, c.Request, indexPath)
	return true
}
