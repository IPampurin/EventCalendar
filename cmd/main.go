package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/EventCalendar/internal/configuration"
	calendarhttp "github.com/IPampurin/EventCalendar/internal/transport/http"
)

func main() {

	// загружаем конфигурацию
	cfg, err := configuration.Load()
	if err != nil {
		slog.Error("ошибка конфигурации", "error", err)
		os.Exit(1)
	}

	// инициализируем логгер
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	// корневой контекст
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем горутину обработки сигналов SIGINT/SIGTERM
	go signalHandler(ctx, cancel)

	// создаем и запускаем HTTP-сервер
	srv := calendarhttp.NewServer(&cfg)
	slog.Info("запуск HTTP", "addr", srv.Addr())
	if err := srv.Run(ctx); err != nil {
		slog.Error("ошибка HTTP-сервера", "error", err)
		os.Exit(1)
	}

	slog.Info("HTTP-сервер корректно остановлен")
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
