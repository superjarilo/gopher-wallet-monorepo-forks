package usecase

import (
	"context"
	"fmt"
)

// Описываем требования к слою данных
type WalletRepo interface {
	ProcessTransaction(ctx context.Context, userID string, amount int64, txType string) error
}

// Описываем требования к брокеру сообщений
type EventPublisher interface {
	PublishTransfer(ctx context.Context, userID string, amount int64, txType string) error
}

type WalletUseCase struct {
	repo      WalletRepo
	publisher EventPublisher
}

func NewWalletUseCase(repo WalletRepo, publisher EventPublisher) *WalletUseCase {
	return &WalletUseCase{repo: repo, publisher: publisher}
}

// Deposit отвечает за зачисление денег
func (uc *WalletUseCase) Deposit(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be greater than zero")
	}

	// Передаем положительное число для увеличения баланса
	if err := uc.repo.ProcessTransaction(ctx, userID, amount, "DEPOSIT"); err != nil {
		return err
	}

	// Шлем событие в Kafka
	return uc.publisher.PublishTransfer(ctx, userID, amount, "DEPOSIT")
}

// Debit отвечает за списание денег
func (uc *WalletUseCase) Debit(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("debit amount must be greater than zero")
	}

	// передаем userID вторым аргументом, как того ожидает репозиторий
	if err := uc.repo.ProcessTransaction(ctx, userID, -amount, "DEBIT"); err != nil {
		return err
	}

	// Шлем событие в Kafka
	return uc.publisher.PublishTransfer(ctx, userID, amount, "DEBIT")
}
