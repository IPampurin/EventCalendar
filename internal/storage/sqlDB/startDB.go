// подключение к Postgres и конструктор хранилища
package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/IPampurin/EventCalendar/internal/configuration"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
)

// Store реализует интерфейс service.EventRepository
type Store struct {
	db *sql.DB
}

// StartDB открывает пул соединений к Postgres через database/sql и драйвер pgx/stdlib
func StartDB(ctx context.Context, cfg *configuration.DBConfig) (*sql.DB, error) {

	dsn := cfg.DSN()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("открытие БД: %w", err)
	}

	// лимиты пула под нагрузку сервиса
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping БД: %w", err)
	}

	return db, nil
}

// NewStore возвращает реализацию service.EventRepository поверх готового *sql.DB
func NewStore(db *sql.DB) *Store {

	return &Store{db: db}
}
