-- +goose Up
-- +goose StatementBegin

-- 1. Таблица кошельков
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,
    balance BIGINT NOT NULL DEFAULT 0, -- Хранение строго в копейках!
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'BLOCKED'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Таблица транзакций (Аудит-лог)
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,            -- Положительное (Deposit), Отрицательное (Debit)
    type VARCHAR(20) NOT NULL,         -- 'DEPOSIT', 'DEBIT'
    status VARCHAR(20) NOT NULL,       -- 'PENDING', 'SUCCESS', 'FAILED'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Высокопроизводительные индексы для Senior-презентации
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_wallets_user_id ON wallets(user_id);

-- Создаем стартового тестового пользователя
INSERT INTO wallets (user_id, balance, currency) VALUES ('user_123', 100000, 'RUB'); -- 1000.00 рублей

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS wallets;
-- +goose StatementEnd
