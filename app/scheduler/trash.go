package scheduler

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"note-vault/app/service"
)

// TrashCleanupScheduler 按固定周期执行到期回收站笔记清理任务。
type TrashCleanupScheduler struct {
	notes    *service.NoteService
	interval time.Duration
	done     chan struct{}
	start    sync.Once
}

// NewTrashCleanupScheduler 创建回收站清理调度器。
func NewTrashCleanupScheduler(notes *service.NoteService, interval time.Duration) (*TrashCleanupScheduler, error) {
	if notes == nil {
		return nil, errors.New("回收站清理任务缺少笔记服务")
	}
	if interval <= 0 {
		return nil, errors.New("回收站清理周期必须大于 0")
	}
	return &TrashCleanupScheduler{
		notes:    notes,
		interval: interval,
		done:     make(chan struct{}),
	}, nil
}

// Start 启动回收站清理任务；服务启动后先执行一次，再按配置周期执行。
func (s *TrashCleanupScheduler) Start(ctx context.Context) {
	s.start.Do(func() {
		go s.run(ctx)
	})
}

// Wait 等待后台清理任务退出，确保数据库连接关闭前当前任务已经结束。
func (s *TrashCleanupScheduler) Wait() {
	<-s.done
}

func (s *TrashCleanupScheduler) run(ctx context.Context) {
	defer close(s.done)

	s.runOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *TrashCleanupScheduler) runOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	startedAt := time.Now()
	scanned, succeeded, failed, err := s.notes.CleanupExpiredTrash(ctx)
	elapsed := time.Since(startedAt)
	if err != nil {
		log.Printf(
			"回收站自动清理失败 scanned=%d succeeded=%d failed=%d elapsed=%s error=%v",
			scanned, succeeded, failed, elapsed, err,
		)
		return
	}
	log.Printf(
		"回收站自动清理完成 scanned=%d succeeded=%d failed=%d elapsed=%s",
		scanned, succeeded, failed, elapsed,
	)
}
