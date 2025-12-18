-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Единая таблица операций по балансу
CREATE TABLE IF NOT EXISTS balance_operations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,             -- Номер заказа (для начисления или списания)
    amount DECIMAL(10,2) NOT NULL,          -- Положительное — начисление, отрицательное — списание
    operation_type TEXT NOT NULL CHECK (operation_type IN ('accrual', 'withdrawal')),
    status TEXT NOT NULL                    -- Статус операции
        CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_balance_operations_user ON balance_operations(user_id);
CREATE INDEX IF NOT EXISTS idx_balance_operations_user_status ON balance_operations(user_id, status);
CREATE INDEX IF NOT EXISTS idx_balance_operations_order ON balance_operations(order_number);
