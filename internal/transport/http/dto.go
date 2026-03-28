// структуры входа-выхода HTTP-слоя
package http

import "encoding/json"

// входящие запросы (тело JSON, Content-Type: application/json)

// CreateEventRequest - тело POST /create_event
type CreateEventRequest struct {
	UserID      int64   `json:"user_id"`               // id пользователя, к которому относится событие
	Title       string  `json:"title"`                 // название события
	Description string  `json:"description,omitempty"` // описание; можно не передавать
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
	UserID int64  `json:"user_id"` // user_id=...
	Date   string `json:"date"`    // date=YYYY-MM-DD - якорная дата для дня/недели/месяца (TZ из конфига)
}

// ответы (обёртка под задание: успех - result, ошибка бизнес-логики - error)

// SuccessResponse - JSON при HTTP 200: {"result": ...}
type SuccessResponse struct {
	Result json.RawMessage `json:"result"` // содержимое зависит от эндпоинта (список событий, id и т.д.)
}

// ErrorResponse - JSON при ошибке: {"error": "..."} (статус по правилам задания)
type ErrorResponse struct {
	Error string `json:"error"` // текст ошибки для клиента
}
