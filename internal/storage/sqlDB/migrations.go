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
	                       updated_at TIMESTAMPTZ NOT NULL);`

	// составной индекс для events
	eventsIndexSchema = `CREATE INDEX IF NOT EXISTS idx_events_user_start
	                         ON events (user_id, start_at);`

	// таблица с архивом событий календаря
	archiveTableSchema = `CREATE TABLE IF NOT EXISTS archive_events (
		                            id UUID PRIMARY KEY,
		                       user_id BIGINT NOT NULL,
		                         title TEXT NOT NULL,
	                       description TEXT NOT NULL DEFAULT '',
		                      start_at TIMESTAMPTZ NOT NULL,
		                        end_at TIMESTAMPTZ, 
		                   reminder_at TIMESTAMPTZ,
		                    created_at TIMESTAMPTZ NOT NULL,
		                    updated_at TIMESTAMPTZ NOT NULL,
		                   archived_at TIMESTAMPTZ NOT NULL);`

	// составной индекс для archive_events
	archiveIndexSchema = `CREATE INDEX IF NOT EXISTS idx_archive_user_archived
		                      ON archive_events (user_id, archived_at);`
)

// Migrations применяет миграции
func Migrations(ctx context.Context, db *sql.DB) error {

	if _, err := db.ExecContext(ctx, eventsTableSchema); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, eventsIndexSchema); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, archiveTableSchema); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, archiveIndexSchema); err != nil {
		return err
	}

	return nil
}
