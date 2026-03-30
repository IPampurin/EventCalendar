package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
)

// nullTime преобразует *time.Time в sql.NullTime
func nullTime(t *time.Time) sql.NullTime {

	if t == nil || t.IsZero() {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

// scanner - интерфейс для sql.Row и sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanEvent сканирует строку из таблицы events в структуру Event (модель БД)
func (s *Store) scanEvent(row scanner) (*Event, error) {

	var e Event
	var endAt, reminderAt sql.NullTime

	err := row.Scan(
		&e.ID,
		&e.UserID,
		&e.Title,
		&e.Description,
		&e.StartAt,
		&endAt,
		&reminderAt,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования события: %w", err)
	}

	// преобразуем sql.NullTime в *time.Time
	if endAt.Valid {
		e.EndAt = &endAt.Time
	}
	if reminderAt.Valid {
		e.ReminderAt = &reminderAt.Time
	}

	return &e, nil
}

// scanArchiveEvent сканирует строку из таблицы archive_events в ArchivEvent
func (s *Store) scanArchiveEvent(row scanner) (*ArchiveEvent, error) {

	var e ArchiveEvent
	var endAt, reminderAt sql.NullTime

	err := row.Scan(
		&e.ID,
		&e.UserID,
		&e.Title,
		&e.Description,
		&e.StartAt,
		&endAt,
		&reminderAt,
		&e.CreatedAt,
		&e.UpdatedAt,
		&e.ArchivedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования архивного события: %w", err)
	}

	if endAt.Valid {
		e.EndAt = &endAt.Time
	}
	if reminderAt.Valid {
		e.ReminderAt = &reminderAt.Time
	}

	return &e, nil
}
