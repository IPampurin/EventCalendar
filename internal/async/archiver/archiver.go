// архиватор старых сообщений
package archiver

import (
	"context"
	"time"

	"github.com/IPampurin/EventCalendar/internal/service"
)

// Archiver реализует service.Archiver - периодическая архивация старых событий
type Archiver struct {
	repo     service.EventRepository
	logger   service.Logger
	interval time.Duration
	ctx      context.Context
}

// NewArchiver возвращает новый архиватор
func NewArchiver(ctx context.Context, repo service.EventRepository, logger service.Logger, interval time.Duration) *Archiver {

	return &Archiver{
		repo:     repo,
		logger:   logger,
		interval: interval,
		ctx:      ctx,
	}
}

// Run - блокируется до отмены ctx, запускает тикер
func (a *Archiver) Run() {

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.archive(); err != nil {
				// при ошибке архивации логируем, но продолжаем выполнение до отмены контекста
				a.logger.Error("ошибка архивации", "error", err)
			}
		}
	}
}

// archive выполняет одну операцию архивации
func (a *Archiver) archive() error {

	now := time.Now().UTC()

	count, err := a.repo.ArchiveOlderThan(a.ctx, now)
	if err != nil {
		return err
	}
	if count > 0 {
		a.logger.Info("архивация выполнена", "archived_count", count)
	}

	return nil
}
