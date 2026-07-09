package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/config"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/pkg/postgres"
)

func main() {
	log.Println("Initializing configuration...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded. App will start on port: %s", cfg.AppPort)

	// Создаем контекст для инициализации базы данных
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	// 1. Безопасно инициализируем пул Postgres, забирая строку подключения СТРОГО из .env
	pgPool, err := postgres.New(initCtx, cfg.DBConn)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer pgPool.Close()
	log.Println("PostgreSQL connection pool initialized successfully.")

	// 2. Вызываем автосгенерированный инжектор Google Wire для сборки слоев.
	// Передаем контекст, пул базы и адрес брокера Kafka СТРОГО из нашего .env конфигуратора.
	appHandler, err := InitializeApplication(initCtx, pgPool.Pool, cfg.KafkaBroker)
	if err != nil {
		log.Fatalf("Failed to initialize application via wire: %v", err)
	}

	// 3. Инициализируем роутер Chi и базовые мидлвары
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Тестовый эндпоинт 
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// 4. КРИТИЧЕСКИЙ ШАГ: Монтируем финтех-маршруты из нашего собранного хендлера!
	// Префикс /api/v1 защищает наше API от конфликтов при будущих обновлениях.
	r.Mount("/api/v1", appHandler.Routes())

	// Настраиваем HTTP-сервер в твоем Senior-стиле
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Канал для прослушивания системных сигналов от ОС
	shutdownSig := make(chan os.Signal, 1)
	signal.Notify(shutdownSig, os.Interrupt, syscall.SIGTERM)

	// Запускаем HTTP-сервер в отдельной горутине
	go func() {
		log.Printf("Starting HTTP server on port %s...", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Listen and serve error: %v", err)
		}
	}()

	// Главный поток блокируется здесь и ждет сигнал выключения от ОС
	<-shutdownSig
	log.Println("Shutdown signal received. Stopping server gracefully...")

	// Создаем контекст с таймаутом на 15 секунд для завершения активных запросов
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped cleanly. Goodbye!")
}
