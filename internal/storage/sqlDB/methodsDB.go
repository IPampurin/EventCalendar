// реализация service.EventRepository для Postgres
package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// Store реализует интерфейс service.EventRepository
type Store struct {
	db *sql.DB
}

// nullTime преобразует *time.Time в sql.NullTime
func nullTime(t *time.Time) sql.NullTime {

	if t == nil || t.IsZero() {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{Time: *t, Valid: true}
}

// scanEvent сканирует строку результата в domain.Event
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanEvent - вспомогательный метод сканирования строки
func (s *Store) scanEvent(row scanner) (*domain.Event, error) {

	var e domain.Event
	var endAt, reminderAt, archivedAt sql.NullTime

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
		&archivedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования события: %w", err)
	}

	if endAt.Valid {
		e.EndAt = &endAt.Time
	}
	if reminderAt.Valid {
		e.ReminderAt = &reminderAt.Time
	}
	if archivedAt.Valid {
		e.ArchivedAt = &archivedAt.Time
	}

	return &e, nil
}

// Create вставляет запись о событии
func (s *Store) Create(ctx context.Context, e *domain.Event) error {

	query := `INSERT INTO events (id,
	                              user_id,
								  title,
								  description,
								  start_at,
								  end_at,
								  reminder_at,
								  created_at,
								  updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.ExecContext(ctx, query,
		e.ID,
		e.UserID,
		e.Title,
		e.Description,
		e.StartAt,
		nullTime(e.EndAt),
		nullTime(e.ReminderAt),
		e.CreatedAt,
		e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ошибка внесения записи о событии: %w", err)
	}

	return nil
}

// Update обновляет запись о событии по id
func (s *Store) Update(ctx context.Context, e *domain.Event) error {

	query := `UPDATE events
		         SET title = $1,
		             description = $2,
		             start_at = $3,
		             end_at = $4,
		             reminder_at = $5,
		             updated_at = $6
		       WHERE id = $7
			         AND user_id = $8
					 AND archived_at IS NULL`

	result, err := s.db.ExecContext(ctx, query,
		e.Title,
		e.Description,
		e.StartAt,
		nullTime(e.EndAt),
		nullTime(e.ReminderAt),
		e.UpdatedAt,
		e.ID,
		e.UserID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления события: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete удаляет событие по id
func (s *Store) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {

	query := `DELETE
	            FROM events
			   WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, eventID, userID)
	if err != nil {
		return fmt.Errorf("ошибка удаления события: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetByID возвращает событие по id
func (s *Store) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {

	query := `SELECT id,
	                 user_id,
					 title,
					 description,
					 start_at,
					 end_at,
					 reminder_at,
					 created_at,
					 updated_at,
					 archived_at
		        FROM events
		       WHERE id = $1
			         AND user_id = $2
					 AND archived_at IS NULL`

	return s.scanEvent(s.db.QueryRowContext(ctx, query, eventID, userID))
}

// ListBetween возвращает события в интервале [start, end)
func (s *Store) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {

	query := `SELECT id,
	                 user_id,
					 title,
					 description,
					 start_at,
					 end_at,
					 reminder_at,
					 created_at,
					 updated_at,
					 archived_at
		        FROM events
		       WHERE user_id = $1
		             AND archived_at IS NULL
		             AND start_at < $3
		             AND (end_at IS NULL OR end_at > $2)
		       ORDER BY start_at`

	rows, err := s.db.QueryContext(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки событий по интервалу: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event, err := s.scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки события: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return events, nil
}

// ArchiveOlderThan помечает события архивными
func (s *Store) ArchiveOlderThan(ctx context.Context, cutoff, mark time.Time) (int, error) {

	query := `UPDATE events
		         SET archived_at = $2,
		             updated_at = $2
		       WHERE archived_at IS NULL 
			     AND ((end_at IS NOT NULL AND end_at < $1) 
				  OR (end_at IS NULL AND start_at < $1))`

	result, err := s.db.ExecContext(ctx, query, cutoff, mark)
	if err != nil {
		return 0, fmt.Errorf("ошибка пометки старых событий архивными: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения количества строк, помеченных архивными: %w", err)
	}

	return int(rows), nil
}
