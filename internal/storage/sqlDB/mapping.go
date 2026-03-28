// маппинг полей БД в domain.Event без отдельной структуры модели строки
package sqldb

import (
	"database/sql"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// eventFromRow собирает сущность из уже распакованных полей одной строки SELECT,
// используется после Scan/Next (не ходит в БД; только nullable-поля в указатели)
func eventFromRow(
	id uuid.UUID,
	userID int64,
	title, description string,
	startAt time.Time,
	endAt, reminderAt, archivedAt sql.NullTime,
	createdAt, updatedAt time.Time,
) *domain.Event {

	e := &domain.Event{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: description,
		StartAt:     startAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if endAt.Valid {
		t := endAt.Time
		e.EndAt = &t
	}
	if reminderAt.Valid {
		t := reminderAt.Time
		e.ReminderAt = &t
	}
	if archivedAt.Valid {
		t := archivedAt.Time
		e.ArchivedAt = &t
	}

	return e
}
