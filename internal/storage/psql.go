package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/jackc/pgx/v5/stdlib" // активация драйвера дл миграции
	"github.com/rs/zerolog/log"
)

type PSQLStorage struct {
	db  *pgxpool.Pool
	dsn string
}

func NewPSQLStorage(ctx context.Context, connString string) (*PSQLStorage, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PSQLStorage{db: pool, dsn: connString}, nil
}

func (s *PSQLStorage) Close() error {
	s.db.Close()
	return nil
}

// --- Balance ---
func (s *PSQLStorage) Migarte(ctx context.Context) error {
	log.Logger.Info().Msg("PSQLStorage.Migrate")

	if err := s.db.Ping(ctx); err != nil {
		log.Logger.Error().Err(err).Msg("Error init migration driver")
		return err
	}

	db, err := sql.Open("pgx", s.dsn)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to open database for migration")
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to ping database")
		return err
	}

	driver, err := pgxMigrate.WithInstance(db, &pgxMigrate.Config{})
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create migrate driver instance")
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"pgx",
		driver,
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create migrate instance")
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Logger.Error().Err(err).Msg("Migration failed")
		return err
	}
	return nil
}

// --- Users ---

func (s *PSQLStorage) SaveUser(ctx context.Context, login, hash string) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
        INSERT INTO users (login, password_hash)
        VALUES ($1, $2)
        ON CONFLICT (login) DO NOTHING
        RETURNING id
    `, login, hash).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PSQLStorage) GetUserByLogin(ctx context.Context, login string) (int64, string, error) {
	var id int64
	var hash string
	err := s.db.QueryRow(ctx, `
        SELECT id, password_hash FROM users WHERE login = $1
    `, login).Scan(&id, &hash)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", ErrUserNotFound
		}
		return 0, "", err
	}
	return id, hash, nil
}

// --- Operations ---

func (s *PSQLStorage) CreateOperation(ctx context.Context, op *models.BalanceOperation) error {
	// Проверка: если это начисление и такой номер уже есть — ошибка
	if op.OperationType == models.AccrualOp {
		var count int
		err := s.db.QueryRow(ctx, `
            SELECT COUNT(*) FROM balance_operations
            WHERE order_number = $1 AND operation_type = 'accrual'
        `, op.OrderNumber).Scan(&count)

		if err != nil {
			return err
		}
		if count > 0 {
			return ErrOrderExists
		}
	}

	// Для списания — проверим баланс
	if op.OperationType == models.WithdrawalOp {
		current, _, err := s.GetBalance(ctx, op.UserID)
		if err != nil {
			return err
		}
		if current < -op.Amount { // op.Amount отрицательное, но -op.Amount = положительная сумма
			return ErrNoMoney
		}
	}

	_, err := s.db.Exec(ctx, `
        INSERT INTO balance_operations (user_id, order_number, amount, operation_type, status, processed_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, op.UserID, op.OrderNumber, op.Amount, string(op.OperationType), op.Status, op.ProcessedAt)

	return err
}

func (s *PSQLStorage) GetOperationsByUser(ctx context.Context, userID int64) ([]*models.BalanceOperation, error) {
	rows, err := s.db.Query(ctx, `
        SELECT order_number, amount, operation_type, status, processed_at
        FROM balance_operations
        WHERE user_id = $1
        ORDER BY processed_at DESC
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []*models.BalanceOperation
	for rows.Next() {
		op := &models.BalanceOperation{}
		var opType string
		if err := rows.Scan(&op.OrderNumber, &op.Amount, &opType, &op.Status, &op.ProcessedAt); err != nil {
			return nil, err
		}
		op.OperationType = models.OperationType(opType)

		// Заполняем JSON-поля в зависимости от типа
		if op.Amount > 0 {
			op.Accrual = op.Amount
		} else if op.Amount < 0 {
			op.Sum = -op.Amount // делаем положительным для вывода
		}
		// если 0 — ничего не заполняем (редко, но возможно)

		ops = append(ops, op)
	}
	return ops, nil
}

// --- Balance ---

func (s *PSQLStorage) GetBalance(ctx context.Context, userID int64) (current, withdrawn float64, err error) {
	err = s.db.QueryRow(ctx, `
        SELECT
            COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
            COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)
        FROM balance_operations
        WHERE user_id = $1 AND status = 'PROCESSED'
    `, userID).Scan(&current, &withdrawn)

	return current, withdrawn, err
}
