package storage

import (
	"context"
	"errors"

	"github.com/JSchatten/go-diploma/internal/models"
)

type Storage interface {
	Close() error

	// Пользователи
	SaveUser(ctx context.Context, login, hash string) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (int64, string, error)

	// Операции
	CreateOperation(ctx context.Context, op *models.BalanceOperation) error
	GetOperationsByUser(ctx context.Context, userID int64) ([]*models.BalanceOperation, error)

	// Заказы
	GetOrder(ctx context.Context, number string) (*models.BalanceOperation, error)

	// Получение
	GetAccrualsByUser(ctx context.Context, userID int64) ([]*models.BalanceOperation, error)
	GetWithdrawalsByUser(ctx context.Context, userID int64) ([]*models.BalanceOperation, error)

	// Баланс
	GetBalance(ctx context.Context, userID int64) (current, withdrawn float64, err error)

	// Для accrual-сервиса
	// заказы со статусом NEW
	GetNewOrders(ctx context.Context) ([]*models.BalanceOperation, error)
	// Обновить статус заказа и начисление
	UpdateOrderStatus(ctx context.Context, orderNumber string, status models.Status, accrual float64) error

	// Миграция
	Migrate(ctx context.Context) error
}

var (
	ErrUserExists    = errors.New("user already exists")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidOrder  = errors.New("invalid order number")
	ErrNoMoney       = errors.New("insufficient funds")
	ErrOrderExists   = errors.New("order already exists")
	ErrOrderMine     = errors.New("order belongs to another user")
	ErrOrderNotFound = errors.New("Order not found")
)
