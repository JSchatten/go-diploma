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
	ErrInvalidOrderFormat = errors.New("invalid order number format")
	ErrOrderExists        = errors.New("order already uploaded")
	ErrOrderBelongsToUser = errors.New("order already uploaded by current user")
)

type OrderService struct {
	storage storage.Storage
}

func NewOrderService(store storage.Storage) *OrderService {
	return &OrderService{storage: store}
}

// UploadOrder загружает номер заказа
func (s *OrderService) UploadOrder(ctx context.Context, userID int64, number string) error {
	if number == "" {
		return errors.New("empty order number")
	}

	if !utils.LuhnCheck(number) {
		return ErrInvalidOrderFormat
	}

	existing, err := s.storage.GetOrder(ctx, number)
	if err != nil && !errors.Is(err, storage.ErrOrderNotFound) {
		return err
	}

	if existing != nil {
		if existing.UserID == userID {
			return ErrOrderBelongsToUser
		}
		return ErrOrderExists
	}

	op := &models.BalanceOperation{
		UserID:        userID,
		OrderNumber:   number,
		Amount:        0,
		OperationType: models.AccrualOp,
		Status:        models.NewStatus,
		ProcessedAt:   time.Now(),
	}

	return s.storage.CreateOperation(ctx, op)
}

// GetOrders возвращает список начислений пользователя
func (s *OrderService) GetOrders(ctx context.Context, userID int64) ([]models.OrderResponse, error) {
	ops, err := s.storage.GetAccrualsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(ops) == 0 {
		return nil, nil
	}

	var result []models.OrderResponse
	for _, op := range ops {
		result = append(result, models.OrderResponse{
			Number:     op.OrderNumber,
			Status:     op.Status,
			Accrual:    op.Accrual,
			UploadedAt: op.ProcessedAt,
		})
	}
	return result, nil
}
