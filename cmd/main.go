package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/EventCalendar/internal/async/archiver"
	"github.com/IPampurin/EventCalendar/internal/async/logger"
	"github.com/IPampurin/EventCalendar/internal/async/scheduler"
	"github.com/IPampurin/EventCalendar/internal/configuration"
	"github.com/IPampurin/EventCalendar/internal/service"
	sqlDB "github.com/IPampurin/EventCalendar/internal/storage/sqlDB"
	calendarhttp "github.com/IPampurin/EventCalendar/internal/transport/http"
)

func main() {

	// загружаем конфигурацию
	cfg, err := configuration.Load()
	if err != nil {
		slog.Error("ошибка конфигурации", "error", err)
		os.Exit(1)
	}

	// инициализируем асинхронный логгер
	appLogger := logger.NewAsyncLogger(cfg.App.LogBufferSize)
	defer appLogger.Close()

	// корневой контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем горутину обработки сигналов SIGINT/SIGTERM
	go signalHandler(ctx, cancel)

	// подключаемся к БД
	db, err := sqlDB.StartDB(ctx, &cfg.DB)
	if err != nil {
		appLogger.Error("ошибка подключения к БД", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// миграции
	if err := sqlDB.Migration(ctx, db); err != nil {
		appLogger.Error("ошибка миграций", "error", err)
		os.Exit(1)
	}

	// репозиторий
	repo := sqlDB.NewStore(db)

	// планировщик напоминаний
	reminderScheduler := scheduler.NewScheduler(appLogger, cfg.App.ReminderQueueSize)
	reminderScheduler.Start()
	defer reminderScheduler.Stop()

	// при старте восстанавливаем ожидающие напоминания (TODO реализовать)
	if err := reminderScheduler.RestorePending(ctx); err != nil {
		appLogger.Error("ошибка восстановления напоминаний", "error", err)
	}

	// сервис календаря
	calendarSvc, err := service.NewCalendarService(repo, reminderScheduler, appLogger, cfg.App.Timezone)
	if err != nil {
		appLogger.Error("ошибка создания сервиса", "error", err)
		os.Exit(1)
	}

	// архиватор
	archiverWorker := archiver.NewArchiver(repo, appLogger, cfg.App.ArchiveEvery, cfg.App.ArchiveOlderThan)
	go func() {
		if err := archiverWorker.Run(ctx); err != nil {
			appLogger.Error("архиватор остановлен с ошибкой", "error", err)
		}
	}()
	defer archiverWorker.Stop()

	// HTTP-сервер
	srv := calendarhttp.NewServer(&cfg, calendarSvc, appLogger)
	appLogger.Info("запуск HTTP", "addr", srv.Addr())

	if err := srv.Run(ctx); err != nil {
		appLogger.Error("ошибка HTTP-сервера", "error", err)
		os.Exit(1)
	}

	appLogger.Info("HTTP-сервер корректно остановлен")
}

// signalHandler обрабатывает сигналы отмены
func signalHandler(ctx context.Context, cancel context.CancelFunc) {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-ctx.Done():
		return
	case <-sigChan:
		cancel()
		return
	}
}
