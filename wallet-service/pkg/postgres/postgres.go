package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres инкапсулирует пул соединений
type Postgres struct {
	Pool *pgxpool.Pool
}

// New Инициализирует пул соединений с конфигурацией 
func New(ctx context.Context, connStr string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse conn string: %w", err)
	}

	// Тонкая настройка пула под нагрузки 
	config.MaxConns = 25                      // Максимум 25 одновременных соединений
	config.MinConns = 5                       // Минимум 5 всегда живых соединений
	config.MaxConnLifetime = time.Hour        // Защита от утечек памяти в долгоживущих соединениях
	config.MaxConnIdleTime = 30 * time.Minute // Закрывать неиспользуемые соединения через полчаса

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	// Обязательный Ping при старте приложения
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}

// Close корректно закрывает пул при Graceful Shutdown
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
