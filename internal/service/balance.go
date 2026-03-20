package service

import (
	"context"
	"errors"
	"time"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/JSchatten/go-diploma/internal/utils"
)

var (
	ErrInvalidSum        = errors.New("sum must be positive")
	ErrInvalidOrder      = errors.New("invalid order number: failed Luhn check")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type BalanceService struct {
	storage storage.Storage
}

func NewBalanceService(store storage.Storage) *BalanceService {
	return &BalanceService{storage: store}
}

// списывает средства, если достаточно баллов
func (s *BalanceService) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if sum <= 0 {
		return ErrInvalidSum
	}

	if !utils.LuhnCheck(order) {
		return ErrInvalidOrder
	}

	current, _, err := s.storage.GetBalance(ctx, userID)
	if err != nil {
		return err
	}

	if current < sum {
		return ErrInsufficientFunds
	}

	op := &models.BalanceOperation{
		UserID:        userID,
		OrderNumber:   order,
		Amount:        -sum,
		OperationType: models.WithdrawalOp,
		Status:        models.ProcessedStatus,
		ProcessedAt:   time.Now(),
	}

	return s.storage.CreateOperation(ctx, op)
}

// текущий баланс
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (float64, float64, error) {
	return s.storage.GetBalance(ctx, userID)
}

func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int64) ([]models.WithdrawalResponse, error) {
	ops, err := s.storage.GetWithdrawalsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(ops) == 0 {
		return nil, nil
	}

	var result []models.WithdrawalResponse
	for _, op := range ops {
		result = append(result, models.WithdrawalResponse{
			Order:       op.OrderNumber,
			Sum:         op.Sum,
			ProcessedAt: op.ProcessedAt,
		})
	}
	return result, nil
}
