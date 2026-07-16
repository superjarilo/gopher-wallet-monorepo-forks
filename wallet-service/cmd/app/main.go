package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

    "google.golang.org/grpc"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/config"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/pkg/postgres"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/pkg/walletgrpc"

	wallethttp "github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/delivery/http"
	walletgrpcserver "github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/delivery/grpc"
)

// Application — контейнер для всех слоев, которые нужно запустить в main
type Application struct {
	HTTPHandler *wallethttp.Handler
	GRPCServer  *walletgrpcserver.Server
}

func NewApplication(httpHandler *wallethttp.Handler, grpcServer *walletgrpcserver.Server) *Application {
	return &Application{
		HTTPHandler: httpHandler,
		GRPCServer:  grpcServer,
	}
}

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

	// 2. Вызываем автосгенерированный инжектор Google Wire для сборки структуры Application.
	// Передаем контекст, пул базы и адрес брокера Kafka СТРОГО из нашего .env конфигуратора.
	app, err := InitializeApplication(initCtx, pgPool.Pool, cfg.KafkaBroker)
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

	// 4. КРИТИЧЕСКИЙ ШАГ: Монтируем HTTP роуты из контейнера app
	// Префикс /api/v1 защищает наше API от конфликтов при будущих обновлениях.
	r.Mount("/api/v1", app.HTTPHandler.Routes())

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

	//======= gRPC-сервер =======	
	// Создаем экземпляр НАШЕГО gRPC-сервера (передаем туда UseCase)
	grpcServer := grpc.NewServer()
	// Регистрируем наш сервер в сгенерированном Protobuf-пакете
	walletgrpc.RegisterWalletServiceServer(grpcServer, app.GRPCServer) 

	// Открываем сетевой порт TCP для gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen tcp for grpc: %v", err)
	}

	// Запускаем gRPC в отдельной фоновой горутине, чтобы он не блокировал HTTP сервер
	go func() {
		log.Println("gRPC Server started on port 50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve error: %v", err)
		}
	}()
	//=============================

	// Главный поток блокируется здесь и ждет сигнал выключения от ОС
	<-shutdownSig
	log.Println("Shutdown signal received. Stopping server gracefully...")

	// Мягко останавливаем gRPC сервер
	log.Println("Stopping gRPC server gracefully...")
	grpcServer.GracefulStop() // Останавливает gRPC сервер красиво и без потери данных
	
	// Мягко останавливаем HTTP сервер
	// Создаем контекст с таймаутом на 15 секунд для завершения активных запросов
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped cleanly. Goodbye!")
}
