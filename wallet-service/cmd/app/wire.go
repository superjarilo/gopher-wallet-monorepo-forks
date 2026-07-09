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
)

// InitializeApplication собирает все зависимости в готовый HTTP-хендлер
func InitializeApplication(ctx context.Context, pool *pgxpool.Pool, kafkaBroker string) (*http.Handler, error) {
	wire.Build(
		// 1. Регистрируем конструкторы инфраструктурного слоя (Репозитории)
		repository.NewWalletRepository,
		repository.NewKafkaPublisher,

		// Связываем конкретные структуры репозиториев с интерфейсами, которые ожидает бизнес-логика
		wire.Bind(new(usecase.WalletRepo), new(*repository.WalletRepository)),
		wire.Bind(new(usecase.EventPublisher), new(*repository.KafkaPublisher)),

		// 2. Регистрируем слой бизнес-логики (Use Cases)
		usecase.NewWalletUseCase,
		wire.Bind(new(http.WalletUseCase), new(*usecase.WalletUseCase)),

		// 3. Регистрируем слой доставки (HTTP-интерфейс)
		http.NewHandler,
	)
	return nil, nil
}

