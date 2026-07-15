//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/delivery/http"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/repository"
	"github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/usecase"

	walletgrpcserver "github.com/aimv/gopher-wallet-monorepo/wallet-service/internal/delivery/grpc"
)

// InitializeApplication собирает все зависимости в структуру Application (в main.go объявлена)
func InitializeApplication(ctx context.Context, pool *pgxpool.Pool, kafkaBroker string) (*Application, error) {
	wire.Build(
		// 1. Инфраструктурный слой данных и брокера
		repository.NewWalletRepository,
		repository.NewKafkaPublisher,
		
		wire.Bind(new(usecase.WalletRepo), new(*repository.WalletRepository)),
		wire.Bind(new(usecase.EventPublisher), new(*repository.KafkaPublisher)),

		// 2. Бизнес-логика (UseCase разделяется между HTTP и gRPC)
		usecase.NewWalletUseCase,

		wire.Bind(new(http.WalletUseCase), new(*usecase.WalletUseCase)),
		wire.Bind(new(walletgrpcserver.WalletUseCase), new(*usecase.WalletUseCase)),

		// 3. Слой доставки (HTTP Хендлер + gRPC Сервер)
		http.NewHandler,
		walletgrpcserver.NewServer,

		// 4. Сборка финального контейнера приложения
		NewApplication,
	)
	return nil, nil
}

