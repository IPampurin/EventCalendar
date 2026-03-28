// миграции схемы Postgres
package sqldb

import (
	"context"
	"database/sql"
)

const (
	// таблица событий календаря
	eventsTableSchema = `CREATE TABLE IF NOT EXISTS events (
	                               id UUID PRIMARY KEY,
	                          user_id BIGINT NOT NULL,
	                            title TEXT NOT NULL,
	                      description TEXT NOT NULL DEFAULT '',
	                         start_at TIMESTAMPTZ NOT NULL,
	                           end_at TIMESTAMPTZ,
	                      reminder_at TIMESTAMPTZ,
	                       created_at TIMESTAMPTZ NOT NULL,
	                       updated_at TIMESTAMPTZ NOT NULL,
	                      archived_at TIMESTAMPTZ
						  );`

	// индекс для events (частичный)
	eventsIndexSchema = `CREATE INDEX IF NOT EXISTS idx_events_user_list
	                         ON events (user_id, start_at)
					      WHERE archived_at IS NULL;`
)

// Migration применяет миграции при старте приложения
func Migration(ctx context.Context, db *sql.DB) error {

	if _, err := db.ExecContext(ctx, eventsTableSchema); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, eventsIndexSchema); err != nil {
		return err
	}

	return nil
}
