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
	older    time.Duration // события старше этого периода уходят в архив
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewArchiver возвращает новый архиватор
func NewArchiver(repo service.EventRepository, logger service.Logger, interval, older time.Duration) *Archiver {

	ctx, cancel := context.WithCancel(context.Background())

	return &Archiver{
		repo:     repo,
		logger:   logger,
		interval: interval,
		older:    older,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Run - блокируется до отмены ctx, запускает тикер
func (a *Archiver) Run(ctx context.Context) error {

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	defer close(a.done)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-a.ctx.Done():
			return nil
		case <-ticker.C:
			if err := a.archive(); err != nil {
				a.logger.Error("ошибка архивации", "error", err)
			}
		}
	}
}

// archive выполняет одну операцию архивации
func (a *Archiver) archive() error {

	cutoff := time.Now().UTC().Add(-a.older)
	mark := time.Now().UTC()

	count, err := a.repo.ArchiveOlderThan(a.ctx, cutoff, mark)
	if err != nil {
		return err
	}
	if count > 0 {
		a.logger.Info("архивация выполнена", "archived_count", count, "cutoff", cutoff)
	}

	return nil
}

// Stop останавливает архиватор
func (a *Archiver) Stop() {

	a.cancel()
	<-a.done
}
