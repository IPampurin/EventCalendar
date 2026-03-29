// реализация service.EventRepository для Postgres
package sqldb

import (
	"context"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/domain"
	"github.com/google/uuid"
)

// Create вставляет активное событие
func (s *Store) Create(ctx context.Context, e *domain.Event) error {

	dbEvent := mapEventToDB(e)
	query := `INSERT INTO events (id,
	                              user_id,
								  title,
								  description, 
								  start_at, end_at, 
								  reminder_at, 
								  created_at, 
								  updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.ExecContext(ctx, query,
		dbEvent.ID,
		dbEvent.UserID,
		dbEvent.Title,
		dbEvent.Description,
		dbEvent.StartAt,
		nullTime(dbEvent.EndAt),
		nullTime(dbEvent.ReminderAt),
		dbEvent.CreatedAt,
		dbEvent.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ошибка вставки события: %w", err)
	}

	return nil
}

// Update обновляет активное событие
func (s *Store) Update(ctx context.Context, e *domain.Event) error {

	dbEvent := mapEventToDB(e)
	query := `UPDATE events
			     SET title = $1,
				     description = $2,
					 start_at = $3,
					 end_at = $4,
					 reminder_at = $5,
					 updated_at = $6
			   WHERE id = $7
			         AND user_id = $8`

	result, err := s.db.ExecContext(ctx, query,
		dbEvent.Title,
		dbEvent.Description,
		dbEvent.StartAt,
		nullTime(dbEvent.EndAt),
		nullTime(dbEvent.ReminderAt),
		dbEvent.UpdatedAt,
		dbEvent.ID,
		dbEvent.UserID,
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

// Delete удаляет активное событие
func (s *Store) Delete(ctx context.Context, userID int64, eventID uuid.UUID) error {

	query := `DELETE 
	            FROM events
			   WHERE id = $1 
			         AND user_id = $2`

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

// GetByID возвращает активное событие по ID
func (s *Store) GetByID(ctx context.Context, userID int64, eventID uuid.UUID) (*domain.Event, error) {

	query := `SELECT id,
	                 user_id, 
					 title, 
					 description, 
					 start_at, 
					 end_at, 
					 reminder_at, 
					 created_at, 
					 updated_at
			    FROM events 
			   WHERE id = $1 
			         AND user_id = $2`

	dbEvent, err := s.scanEvent(s.db.QueryRowContext(ctx, query, eventID, userID))
	if err != nil {
		return nil, err
	}

	return mapDBToEvent(*dbEvent), nil
}

// ListBetween возвращает активные события пользователя за интервал [start, end) UTC
func (s *Store) ListBetween(ctx context.Context, userID int64, start, end time.Time) ([]*domain.Event, error) {

	query := `SELECT id, 
	                 user_id, title, 
					 description, 
					 start_at, 
					 end_at, 
					 reminder_at, 
					 created_at, 
					 updated_at
			    FROM events
			   WHERE user_id = $1 
			         AND start_at < $3 
					 AND (end_at IS NULL OR end_at > $2)
			   ORDER BY start_at`

	rows, err := s.db.QueryContext(ctx, query, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки событий: %w", err)
	}
	defer rows.Close()

	events := make([]*domain.Event, 0)
	for rows.Next() {
		dbEvent, err := s.scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		events = append(events, mapDBToEvent(*dbEvent))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации: %w", err)
	}

	return events, nil
}
