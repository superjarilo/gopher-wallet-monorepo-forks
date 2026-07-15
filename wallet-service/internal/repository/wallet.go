package repository

import (
	"context"
	"fmt"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepository struct {
	pool    *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{
		pool:    pool,
		// Настраиваем squirrel под плейсхолдеры PostgreSQL ($1, $2)
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// ProcessTransaction выполняет изменение баланса и аудит в одной транзакции
func (r *WalletRepository) ProcessTransaction(ctx context.Context, userID string, amount int64, txType string) error {
	// 1. Открываем транзакцию
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	// Если произойдет ошибка или паника, defer автоматически сделает Rollback.
	// Если транзакция закоммичена, Rollback безопасно проигнорируется.
	defer tx.Rollback(ctx)

	// 2. Блокируем строку кошелька (SELECT ... FOR UPDATE) и проверяем баланс
	var walletID string
	var currentBalance int64
	var status string

	querySelect, argsSelect, err := r.builder.
		Select("id", "balance", "status").
		From("wallets").
		Where(squirrel.Eq{"user_id": userID}).
		Suffix("FOR UPDATE"). // Критично для финтеха: блокируем строку от параллельных изменений
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build select sql: %w", err)
	}

	err = tx.QueryRow(ctx, querySelect, argsSelect...).Scan(&walletID, &currentBalance, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("wallet for user %s not found", userID)
		}
		return fmt.Errorf("failed to select wallet: %w", err)
	}

	// Проверка на блокировку аккаунта (антифрод)
	if status == "BLOCKED" {
		return fmt.Errorf("operation forbidden: wallet is blocked")
	}

	// Если идет списание (DEBIT), amount у нас будет отрицательным. Проверяем баланс.
	if txType == "DEBIT" && currentBalance+amount < 0 {
		return fmt.Errorf("insufficient funds: balance is %d, requested %d", currentBalance, -amount)
	}

	// 3. Обновляем текущий баланс кошелька
	queryUpdate, argsUpdate, err := r.builder.
		Update("wallets").
		Set("balance", squirrel.Expr("balance + ?", amount)).
		Where(squirrel.Eq{"id": walletID}).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build update sql: %w", err)
	}

	_, err = tx.Exec(ctx, queryUpdate, argsUpdate...)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// 4. Записываем финансовый след в таблицу transactions
	queryInsert, argsInsert, err := r.builder.
		Insert("transactions").
		Columns("wallet_id", "amount", "type", "status").
		Values(walletID, amount, txType, "SUCCESS").
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build insert sql: %w", err)
	}

	_, err = tx.Exec(ctx, queryInsert, argsInsert...)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	// 5. Сохраняем транзакцию в БД навсегда
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Получение данных кошелька пользователя из Postgres
func (r *WalletRepository) GetByUserID(ctx context.Context, userID string) (string, int64, string, string, error) {
	// 1. Строим чистый SQL через Squirrel
	query, args, err := r.builder.
		Select("id", "balance", "status", "currency").
		From("wallets").
		Where(squirrel.Eq{"user_id": userID}).
		ToSql()

	if err != nil {
		return "", 0, "", "", fmt.Errorf("failed to build select sql: %w", err)
	}

	// 2. Выполняем запрос к пулу pgx, передавая context для поддержки таймаутов
	var id, status, currency string
	var balance int64

	err = r.pool.QueryRow(ctx, query, args...).Scan(&id, &balance, &status, &currency)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", 0, "", "", fmt.Errorf("wallet not found for user_id: %s", userID)
		}
		return "", 0, "", "", fmt.Errorf("database query failed: %w", err)
	}

	return id, balance, status, currency, nil
}
