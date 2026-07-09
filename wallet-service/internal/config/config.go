package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string `env:"APP_PORT" envDefault:"8080"`
	DBHost      string `env:"DB_HOST" envDefault:"localhost"`
	DBPort      string `env:"DB_PORT" envDefault:"5432"`
	DBUser      string `env:"DB_USER" envDefault:"postgres"`
	DBPassword  string `env:"DB_PASSWORD"`
	DBName      string `env:"DB_NAME" envDefault:"wallet_db"`
	DBConn      string `env:"DB_CONN" envDefault:"postgres://wallet_user:wallet_password@localhost:5432/wallet_db?sslmode=disable"`
	KafkaBroker string `env:"KAFKA_BROKER" envDefault:"localhost:9092"`
}

func Load() (*Config, error) {
	// Подгружаем локальный .env файл, если он есть
	godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
