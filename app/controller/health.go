package controller

import (
	"context"
	"errors"
	"net/http"
	"note-vault/app/utils"

	"note-vault/app/config"

	"github.com/gin-gonic/gin"
)

type healthData struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
}

// Live 表示 HTTP 进程仍可正常接收和处理请求。
func Live(c *gin.Context) {
	utils.Success(c, healthData{
		Status: "UP",
		Components: map[string]string{
			"http": "UP",
		},
	})
}

// Ready 检查 PostgreSQL 和 Redis 是否可用。
func Ready(c *gin.Context) {
	components := map[string]string{
		"postgresql": "UP",
		"redis":      "UP",
	}

	ready := true
	if err := checkPostgreSQL(c.Request.Context()); err != nil {
		components["postgresql"] = "DOWN"
		ready = false
	}
	if err := checkRedis(c.Request.Context()); err != nil {
		components["redis"] = "DOWN"
		ready = false
	}

	data := healthData{Status: "UP", Components: components}
	if !ready {
		data.Status = "DOWN"
		utils.ErrorWithData(
			c,
			http.StatusServiceUnavailable,
			utils.CodeServiceUnavailable,
			"服务尚未就绪",
			data,
		)
		return
	}

	utils.Success(c, data)
}

func checkPostgreSQL(parent context.Context) error {
	if config.DB == nil {
		return errors.New("PostgreSQL 客户端未初始化")
	}

	database, err := config.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parent, config.AppConfig.Server.HealthCheckTimeoutDuration())
	defer cancel()
	return database.PingContext(ctx)
}

func checkRedis(parent context.Context) error {
	if config.Redis == nil {
		return errors.New("Redis 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(parent, config.AppConfig.Server.HealthCheckTimeoutDuration())
	defer cancel()
	return config.Redis.Ping(ctx).Err()
}
