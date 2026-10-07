package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"note-vault/app/config"
	"note-vault/app/router"
	"note-vault/app/scheduler"
	"note-vault/app/service"
)

const dependencyInitTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Printf("服务退出: %v", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	if err := config.LoadConfig(); err != nil {
		return err
	}

	defer func() {
		runErr = errors.Join(runErr, config.CloseClients())
	}()

	if err := initDependencies(); err != nil {
		return err
	}

	engine, err := router.InitRouter()
	if err != nil {
		return err
	}

	// 初始化定时任务
	trashCleanupScheduler, err := scheduler.NewTrashCleanupScheduler(
		service.NewNoteService(),
		config.AppConfig.Trash.CleanupIntervalDuration(),
	)
	if err != nil {
		return fmt.Errorf("初始化回收站清理任务失败: %w", err)
	}
	cleanupContext, cancelCleanup := context.WithCancel(context.Background())
	defer func() {
		cancelCleanup()
		trashCleanupScheduler.Wait()
	}()
	trashCleanupScheduler.Start(cleanupContext)

	readTimeout, _ := time.ParseDuration(config.AppConfig.Server.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(config.AppConfig.Server.WriteTimeout)
	shutdownTimeout, _ := time.ParseDuration(config.AppConfig.Server.ShutdownTimeout)

	server := &http.Server{
		Addr:         config.AppConfig.ServerAddress(),
		Handler:      engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	serverResult := make(chan error, 1)
	go func() {
		log.Printf("HTTP 服务启动，监听地址 %s", server.Addr)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			serverResult <- nil
			return
		}
		serverResult <- err
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverResult:
		if err != nil {
			return fmt.Errorf("HTTP 服务运行失败: %w", err)
		}
		return nil
	case <-signalContext.Done():
		log.Println("收到退出信号，开始关闭 HTTP 服务")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("关闭 HTTP 服务失败: %w", err)
	}

	if err := <-serverResult; err != nil {
		return fmt.Errorf("HTTP 服务关闭失败: %w", err)
	}

	log.Println("HTTP 服务已停止")
	return nil
}

func initDependencies() error {
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), dependencyInitTimeout)
	databaseErr := config.InitDB(databaseContext)
	cancelDatabase()
	if databaseErr != nil {
		return databaseErr
	}

	redisContext, cancelRedis := context.WithTimeout(context.Background(), dependencyInitTimeout)
	redisErr := config.InitRedis(redisContext)
	cancelRedis()
	if redisErr != nil {
		return redisErr
	}

	return nil
}
