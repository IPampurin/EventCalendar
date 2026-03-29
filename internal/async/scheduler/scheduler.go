// напоминальщик
package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/google/uuid"
)

// Scheduler реализует service.ReminderScheduler через канал и таймеры
type Scheduler struct {
	queue  chan domain.ReminderTask
	logger service.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	timers map[uuid.UUID]*time.Timer // для отмены
}

// NewScheduler возвращает новый напоминатель о событиях
func NewScheduler(logger service.Logger, queueSize int) *Scheduler {

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		queue:  make(chan domain.ReminderTask, queueSize),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		timers: make(map[uuid.UUID]*time.Timer),
	}
}

// Start запускает фоновую горутину, обрабатывающую задачи
func (s *Scheduler) Start() {

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		for {
			select {
			case <-s.ctx.Done():
				return
			case task, ok := <-s.queue:
				if !ok {
					return
				}
				s.scheduleTask(task)
			}
		}
	}()
}

// scheduleTask создаёт таймер на отправку напоминания
func (s *Scheduler) scheduleTask(task domain.ReminderTask) {

	delay := time.Until(task.RemindAt)
	if delay < 0 {
		delay = 0
	}
	timer := time.NewTimer(delay)

	// сохраняем таймер для возможной отмены
	s.mu.Lock()
	s.timers[task.EventID] = timer
	s.mu.Unlock()

	go func() {
		select {
		case <-timer.C:
			// отправляем уведомление (пока только логируем)
			s.logger.Info("напоминание сработало",
				"event_id", task.EventID,
				"user_id", task.UserID,
				"title", task.Title,
			)
			s.mu.Lock()
			delete(s.timers, task.EventID)
			s.mu.Unlock()
		case <-s.ctx.Done():
			// Остановка — таймер не сработает
			if !timer.Stop() {
				<-timer.C
			}
			s.mu.Lock()
			delete(s.timers, task.EventID)
			s.mu.Unlock()
		}
	}()
}

// Schedule добавляет задачу в очередь
func (s *Scheduler) Schedule(ctx context.Context, task domain.ReminderTask) error {

	select {
	case s.queue <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cancel отменяет запланированное напоминание
func (s *Scheduler) Cancel(ctx context.Context, eventID uuid.UUID) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if timer, ok := s.timers[eventID]; ok {
		if !timer.Stop() {
			<-timer.C
		}
		delete(s.timers, eventID)
	}

	return nil
}

// RestorePending - заглушка (TODO: загружает из БД будущие напоминания)
func (s *Scheduler) RestorePending(ctx context.Context) error {

	// можно реализовать вызов репозитория и повторную постановку задач

	return nil
}

// Stop останавливает воркер и отменяет все таймеры
func (s *Scheduler) Stop() {

	s.cancel()
	close(s.queue) // закрываем канал, чтобы горутина завершилась
	s.wg.Wait()
}
