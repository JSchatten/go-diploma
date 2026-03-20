package models

import "time"

// Вобще хотелось бы в протобафе это всё

// OperationType тип операции
type OperationType string

const (
	AccrualOp    OperationType = "accrual"
	WithdrawalOp OperationType = "withdrawal"
)

// Status статус обработки операции
type Status string

const (
	NewStatus        Status = "NEW"
	ProcessingStatus Status = "PROCESSING"
	InvalidStatus    Status = "INVALID"
	ProcessedStatus  Status = "PROCESSED"
)

// BalanceOperation — внутренняя модель операции в БД
type BalanceOperation struct {
	ID            int64         `json:"-"`           // не в JSON
	UserID        int64         `json:"-"`           // не в JSON
	OrderNumber   string        `json:"-"`           // не в JSON напрямую
	Amount        float64       `json:"-"`           // хранит знак: + для начислений, - для списаний
	OperationType OperationType `json:"-"`           // тип операции
	Status        Status        `json:"status"`      // статус
	ProcessedAt   time.Time     `json:"uploaded_at"` // RFC3339

	//  только для JSON-сериализации
	Accrual float64 `json:"accrual,omitempty"` // только если начисление > 0
	Sum     float64 `json:"sum,omitempty"`     // только если списание
}

// GET /api/user/orders
type OrderResponse struct {
	Number     string    `json:"number"`
	Status     Status    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// GET /api/user/withdrawals
type WithdrawalResponse struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

// для POST /api/user/balance/withdraw
type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}
