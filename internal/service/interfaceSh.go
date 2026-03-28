// интерфейс напоминальщика - порт планировщика (канал, брокер и т.д.)
package service

import (
	"context"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// ReminderScheduler - постановка и отмена напоминаний без деталей реализации очереди
type ReminderScheduler interface {
	
	// Schedule - зарегистрировать отправку напоминания в remindAt (идемпотентно по eventID при необходимости)
	Schedule(ctx context.Context, task domain.ReminderTask) error

	// Cancel - снять напоминание для события (при удалении события или снятии reminder)
	Cancel(ctx context.Context, eventID uuid.UUID) error

	// RestorePending - поднять из БД несработанные будущие напоминания после рестарта (опционально для реализации)
	RestorePending(ctx context.Context) error
}
