// напоминальщик
package scheduler

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/google/uuid"
)

// Scheduler управляет очередью напоминаний через priority queue
type Scheduler struct {
	repo       service.EventRepository
	log        service.Logger
	addChan    chan domain.ReminderTask
	cancelChan chan uuid.UUID
	ctx        context.Context // корневой контекст из main
}

// taskQueue — priority queue (минимум по RemindAt)
type taskQueue []domain.ReminderTask

func (tq taskQueue) Len() int           { return len(tq) }
func (tq taskQueue) Less(i, j int) bool { return tq[i].RemindAt.Before(tq[j].RemindAt) }
func (tq taskQueue) Swap(i, j int)      { tq[i], tq[j] = tq[j], tq[i] }

func (tq *taskQueue) Push(x interface{}) {
	*tq = append(*tq, x.(domain.ReminderTask))
}

func (tq *taskQueue) Pop() interface{} {
	old := *tq
	n := len(old)
	item := old[n-1]
	*tq = old[:n-1]
	return item
}

// NewScheduler создаёт планировщик с корневым контекстом
func NewScheduler(ctx context.Context, repo service.EventRepository, log service.Logger, bufSize int) *Scheduler {
	return &Scheduler{
		repo:       repo,
		log:        log,
		addChan:    make(chan domain.ReminderTask, bufSize),
		cancelChan: make(chan uuid.UUID, bufSize),
		ctx:        ctx,
	}
}

// Run блокируется до отмены контекста
func (s *Scheduler) Run() {
	now := time.Now().UTC()

	// Загружаем из БД будущие напоминания
	events, err := s.repo.GetPendingReminders(s.ctx, now)
	if err != nil {
		s.log.Error("ошибка загрузки напоминаний из БД", "error", err)
	}

	pq := &taskQueue{}
	heap.Init(pq)

	for _, e := range events {
		if e.ReminderAt != nil && e.ReminderAt.After(time.Now().UTC()) {
			heap.Push(pq, domain.ReminderTask{
				EventID:  e.ID,
				UserID:   e.UserID,
				RemindAt: *e.ReminderAt,
				Title:    e.Title,
			})
		}
	}

	s.log.Info("планировщик запущен", "loaded", len(events), "queue_len", pq.Len())

	for {
		// Очередь пуста — ждём только добавления или отмены контекста
		if pq.Len() == 0 {
			select {
			case <-s.ctx.Done():
				return
			case task := <-s.addChan:
				if task.RemindAt.After(time.Now().UTC()) {
					heap.Push(pq, task)
				}
			case id := <-s.cancelChan:
				s.remove(pq, id)
			}
			continue
		}

		// Есть задачи — ждём либо ближайшую, либо новое событие
		next := (*pq)[0]
		wait := time.Until(next.RemindAt)

		if wait <= 0 {
			// Время уже пришло
			heap.Pop(pq)
			s.fire(next)
			continue
		}

		timer := time.NewTimer(wait)
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			heap.Pop(pq)
			s.fire(next)
		case task := <-s.addChan:
			timer.Stop()
			if task.RemindAt.After(time.Now().UTC()) {
				heap.Push(pq, task)
			}
		case id := <-s.cancelChan:
			timer.Stop()
			s.remove(pq, id)
		}
	}
}

// Schedule добавляет задачу в канал (неблокирующе, с проверкой контекста)
func (s *Scheduler) Schedule(ctx context.Context, task domain.ReminderTask) error {
	select {
	case s.addChan <- task:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("контекст отменён")
	case <-s.ctx.Done():
		return fmt.Errorf("планировщик остановлен")
	default:
		return fmt.Errorf("канал переполнен")
	}
}

// Cancel удаляет задачу из очереди
func (s *Scheduler) Cancel(ctx context.Context) {

}
