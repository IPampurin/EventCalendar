// напоминальщик с очередью задач и восстановлением из БД
package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/IPampurin/EventCalendar/internal/service"
	"github.com/google/uuid"
)

// taskQueue - priority queue (минимум по RemindAt)
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

// remove удаляет задачу по eventID из очереди (линейный поиск, но очередь небольшая)
func (tq *taskQueue) remove(eventID uuid.UUID) bool {

	for i, task := range *tq {
		if task.EventID == eventID {
			heap.Remove(tq, i)
			return true
		}
	}

	return false
}

// Scheduler управляет очередью напоминаний через priority queue
type Scheduler struct {
	repo       service.EventRepository
	log        service.Logger
	addChan    chan domain.ReminderTask
	cancelChan chan uuid.UUID
	ctx        context.Context
	wg         sync.WaitGroup
	// буфер для задач, которые не удалось отправить в addChan (DLQ)
	pendingTasks []domain.ReminderTask
	pendingMu    sync.Mutex
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

// Run блокируется до отмены контекста, запускает основную логику
func (s *Scheduler) Run() {

	s.wg.Add(1)
	defer s.wg.Done()

	// восстанавливаем будущие напоминания из БД
	now := time.Now().UTC()

	events, err := s.repo.GetPendingReminders(s.ctx, now)
	if err != nil {
		s.log.Error("ошибка загрузки напоминаний из БД", "error", err)
	}

	pq := &taskQueue{}
	heap.Init(pq)

	for _, e := range events {
		if e.ReminderAt != nil && e.ReminderAt.After(now) {
			heap.Push(pq, domain.ReminderTask{
				EventID:  e.ID,
				UserID:   e.UserID,
				RemindAt: *e.ReminderAt,
				Title:    e.Title,
			})
		}
	}

	s.log.Info("планировщик запущен", "loaded", len(events), "queue_len", pq.Len())

	// вспомогательная функция для обработки добавления задачи
	addTask := func(task domain.ReminderTask) {
		if task.RemindAt.After(time.Now().UTC()) {
			heap.Push(pq, task)
			s.log.Debug("задача добавлена в очередь", "event_id", task.EventID, "remind_at", task.RemindAt)
		} else {
			s.log.Info("попытка добавить просроченное напоминание", "event_id", task.EventID)
		}
	}

	for {
		if pq.Len() == 0 {
			// очередь пуста - ждём только добавления или отмены
			select {
			case <-s.ctx.Done():
				s.drainPendingTasks()
				return
			case task := <-s.addChan:
				addTask(task)
			case id := <-s.cancelChan:
				if pq.remove(id) {
					s.log.Debug("задача отменена", "event_id", id)
				}
			}
			continue
		}

		next := (*pq)[0]
		wait := time.Until(next.RemindAt)

		if wait <= 0 {
			// время уже пришло или прошло
			heap.Pop(pq)
			s.send(next)
			continue
		}

		timer := time.NewTimer(wait)
		select {
		case <-s.ctx.Done():
			timer.Stop()
			s.drainPendingTasks()
			return
		case <-timer.C:
			heap.Pop(pq)
			s.send(next)
		case task := <-s.addChan:
			timer.Stop()
			addTask(task)
		case id := <-s.cancelChan:
			timer.Stop()
			if pq.remove(id) {
				s.log.Debug("задача отменена", "event_id", id)
			}
		}
	}
}

// send отправляет напоминание (логирует)
func (s *Scheduler) send(task domain.ReminderTask) {

	s.log.Info("напоминание сработало",
		"event_id", task.EventID,
		"user_id", task.UserID,
		"title", task.Title,
	)
	// TODO: реальная отправка (например, HTTP, email, tg)
}

// Schedule добавляет задачу в канал с повторными попытками
func (s *Scheduler) Schedule(ctx context.Context, task domain.ReminderTask) error {

	const maxRetries = 3
	const retryDelay = 10 * time.Millisecond

	// проверяем контекст или возможность немедленной отправки
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.addChan <- task:
		return nil
	default:
		for attempt := 0; attempt < maxRetries; attempt++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case s.addChan <- task:
				return nil
			default:
				// канал переполнен, ждём немного и пробуем снова
				if attempt < maxRetries-1 {
					time.Sleep(retryDelay)
					continue
				}
				// последняя попытка не удалась - сохраняем в отложенный буфер (DLQ)
				s.pendingMu.Lock()
				s.pendingTasks = append(s.pendingTasks, task)
				s.pendingMu.Unlock()
				s.log.Error("канал напоминаний переполнен после ретраев, задача сохранена в буфер",
					"event_id", task.EventID, "attempts", maxRetries)
				return nil
			}
		}
	}

	return nil
}

// Cancel отменяет задачу по eventID (без ретраев, т.к. операция редкая)
func (s *Scheduler) Cancel(ctx context.Context, eventID uuid.UUID) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		select {
		case s.cancelChan <- eventID:
			return nil
		default:
			s.log.Error("канал отмены переполнен", "event_id", eventID)
			return nil
		}
	}
}

// drainPendingTasks перед остановкой пытается обработать отложенные задачи (выводит в лог)
func (s *Scheduler) drainPendingTasks() {

	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	if len(s.pendingTasks) == 0 {
		return
	}

	// для примера просто выводим в лог импровизированную DLQ
	s.log.Info("остановка планировщика, необработанные задачи в буфере", "count", len(s.pendingTasks))
	for _, task := range s.pendingTasks {
		s.log.Error("потерянное напоминание", "event_id", task.EventID, "user_id", task.UserID, "remind_at", task.RemindAt)
	}
}
