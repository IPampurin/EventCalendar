// структуры входа-выхода HTTP-слоя
package http

import (
	"encoding/json"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
)

// входящие запросы (тело JSON, Content-Type: application/json)

// CreateEventRequest - тело POST /create_event
type CreateEventRequest struct {
	UserID      int64   `json:"user_id"`               // id пользователя, к которому относится событие
	Title       string  `json:"title"`                 // название события
	Description string  `json:"description,omitempty"` // описание
	StartAt     string  `json:"start_at"`              // начало события RFC3339 ("2026-03-27T10:00:00+03:00")
	EndAt       *string `json:"end_at,omitempty"`      // окончание события RFC3339 или null (точечное событие)
	ReminderAt  *string `json:"reminder_at,omitempty"` // момент, когда будет отправлено напоминание RFC3339 или null (без напоминания)
}

// UpdateEventRequest - тело POST /update_event
type UpdateEventRequest struct {
	UserID      int64   `json:"user_id"`               // id пользователя
	EventID     string  `json:"event_id"`              // uid события (строка UUID)
	Title       string  `json:"title"`                 // новое название
	Description string  `json:"description,omitempty"` // новое описание
	StartAt     string  `json:"start_at"`              // новое начало: RFC3339
	EndAt       *string `json:"end_at,omitempty"`      // новое окончание или null
	ReminderAt  *string `json:"reminder_at,omitempty"` // новый момент напоминания; null - снять напоминание
}

// DeleteEventRequest - тело POST /delete_event
type DeleteEventRequest struct {
	UserID  int64  `json:"user_id"`  // id пользователя
	EventID string `json:"event_id"` // uid удаляемого события
}

// EventsForPeriodQuery - параметры GET /events_for_day|week|month (query string)
type EventsForPeriodQuery struct {
	UserID int64  `form:"user_id"` // user_id=...
	Date   string `form:"date"`    // date=YYYY-MM-DD - якорная дата для дня/недели/месяца (TZ из конфига)
}

// ответы (обёртка под задание: успех - result, ошибка бизнес-логики - error)

// SuccessResponse - JSON при HTTP 200: {"result": ...}
type SuccessResponse struct {
	Result json.RawMessage `json:"result"` // содержимое зависит от эндпоинта (список событий, id и т.д.)
}

// ErrorResponse - JSON при ошибке: {"error": "..."}
type ErrorResponse struct {
	Error string `json:"error"` // текст ошибки для клиента
}

// EventResponse - структура для ответа фронту
type EventResponse struct {
	ID          string     `json:"id"`
	UserID      int64      `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	StartAt     time.Time  `json:"start_at"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	ReminderAt  *time.Time `json:"reminder_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// toEventResponse преобразует доменную модель в модель dto
func toEventResponse(e *domain.Event) EventResponse {

	return EventResponse{
		ID:          e.ID.String(),
		UserID:      e.UserID,
		Title:       e.Title,
		Description: e.Description,
		StartAt:     e.StartAt,
		EndAt:       e.EndAt,
		ReminderAt:  e.ReminderAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		ArchivedAt:  e.ArchivedAt,
	}
}

// toEventResponses преобразует слайс доменных моделей в слайс моделей dto
func toEventResponses(events []*domain.Event) []EventResponse {

	result := make([]EventResponse, len(events))
	for i, e := range events {
		result[i] = toEventResponse(e)
	}

	return result
}
