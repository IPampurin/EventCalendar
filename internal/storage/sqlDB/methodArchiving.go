package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ArchiveOlderThan переносит прошедшие (StartAt, при EndAt == nil) и закончившиеся (EndAt) события в архив
// и удаляет их из активной таблицы (транзакция), mark - время начала процедуры архивации (записывается в archived_at),
// возвращает количество заархивированных событий
func (s *Store) ArchiveOlderThan(ctx context.Context, mark time.Time) (int, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции архивации: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// выбираем старые события
	selectQuery := `SELECT id, 
	                       user_id, 
						   title, 
						   description, 
						   start_at, 
						   end_at, 
						   reminder_at, 
						   created_at, 
						   updated_at
					  FROM events
					 WHERE (end_at IS NOT NULL AND end_at < $1) 
					       OR (end_at IS NULL AND start_at < $1)`

	rows, err := tx.QueryContext(ctx, selectQuery, mark)
	if err != nil {
		return 0, fmt.Errorf("ошибка выборки старых событий: %w", err)
	}
	defer rows.Close()

	var archivedCount int
	for rows.Next() {

		var dbEvent Event
		var endAt, reminderAt sql.NullTime
		if err := rows.Scan(
			&dbEvent.ID,
			&dbEvent.UserID,
			&dbEvent.Title,
			&dbEvent.Description,
			&dbEvent.StartAt,
			&endAt,
			&reminderAt,
			&dbEvent.CreatedAt,
			&dbEvent.UpdatedAt); err != nil {
			return 0, fmt.Errorf("ошибка сканирования строки при архивации: %w", err)
		}
		if endAt.Valid {
			dbEvent.EndAt = &endAt.Time
		}
		if reminderAt.Valid {
			dbEvent.ReminderAt = &reminderAt.Time
		}
		// копируем в архив
		archEvent := ArchiveEvent{
			ID:          dbEvent.ID,
			UserID:      dbEvent.UserID,
			Title:       dbEvent.Title,
			Description: dbEvent.Description,
			StartAt:     dbEvent.StartAt,
			EndAt:       dbEvent.EndAt,
			ReminderAt:  dbEvent.ReminderAt,
			CreatedAt:   dbEvent.CreatedAt,
			UpdatedAt:   dbEvent.UpdatedAt,
			ArchivedAt:  mark,
		}

		insertQuery := `INSERT INTO archive_events (id, 
		                                            user_id, 
													title, 
													description, 
													start_at, 
													end_at, 
													reminder_at, 
													created_at, 
													updated_at, 
													archived_at)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

		_, err = tx.ExecContext(ctx, insertQuery,
			archEvent.ID,
			archEvent.UserID,
			archEvent.Title,
			archEvent.Description,
			archEvent.StartAt,
			archEvent.EndAt,
			archEvent.ReminderAt,
			archEvent.CreatedAt,
			archEvent.UpdatedAt,
			archEvent.ArchivedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("ошибка вставки в архив: %w", err)
		}
		archivedCount++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	// удаляем заархивированные события из активной таблицы
	if archivedCount > 0 {
		deleteQuery := `DELETE 
		                  FROM events 
						 WHERE (end_at IS NOT NULL AND end_at < $1) 
						       OR (end_at IS NULL AND start_at < $1)`

		_, err = tx.ExecContext(ctx, deleteQuery, mark)
		if err != nil {
			return 0, fmt.Errorf("ошибка удаления старых событий: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка коммита транзакции архивации: %w", err)
	}

	return archivedCount, nil
}
