package accrual

import (
	"context"
	"fmt"
	"time"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

const (
	PollInterval = 1 * time.Second
	HTTPTimeout  = 5 * time.Second
)

type Client struct {
	client     *resty.Client
	accrualURL string
	storage    storage.Storage
}

func NewClient(accrualURL string, store storage.Storage) *Client {
	client := resty.New()
	client.SetTimeout(HTTPTimeout)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return r.StatusCode() == 429 || // Too Many Requests
			r.StatusCode() >= 500 || // Server error
			err != nil // Network error
	})

	return &Client{
		client:     client,
		accrualURL: accrualURL,
		storage:    store,
	}
}

// StartPolling запускает фоновый опрос статусов
func (c *Client) StartPolling(ctx context.Context) {
	log.Info().Msg("Starting accrual status poller")

	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Shutting down accrual poller")
			return
		case <-ticker.C:
			c.pollOrders()
		}
	}
}

// pollOrders находит заказы со статусом NEW и проверяет их у accrual
func (c *Client) pollOrders() {
	orders, err := c.storage.GetNewOrders(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to get new orders")
		return
	}

	for _, order := range orders {
		status, accrualAmount, err := c.fetchOrderStatus(order.OrderNumber)
		if err != nil {
			log.Warn().Err(err).Str("order", order.OrderNumber).Msg("Failed to fetch order status")
			continue
		}

		if err := c.storage.UpdateOrderStatus(
			context.Background(),
			order.OrderNumber,
			status,
			accrualAmount,
		); err != nil {
			log.Error().Err(err).Str("order", order.OrderNumber).Msg("Failed to update order status")
			continue
		}

		log.Info().
			Str("order", order.OrderNumber).
			Str("status", string(status)).
			Float64("accrual", accrualAmount).
			Msg("Order status updated")
	}
}

// fetchOrderStatus запрашивает статус у accrual-сервиса
func (c *Client) fetchOrderStatus(orderNumber string) (models.Status, float64, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.accrualURL, orderNumber)

	var response struct {
		Order   string  `json:"order"`
		Status  string  `json:"status"`
		Accrual float64 `json:"accrual,omitempty"`
	}

	resp, err := c.client.R().
		SetResult(&response).
		Get(url)

	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}

	// Обработка статусов
	if resp.StatusCode() == 204 {
		return models.NewStatus, 0, nil
	}

	if resp.StatusCode() != 200 {
		return "", 0, fmt.Errorf("unexpected status: %d", resp.StatusCode())
	}

	status := models.Status(response.Status)

	switch status {
	case models.NewStatus, models.ProcessingStatus, models.InvalidStatus, models.ProcessedStatus:
		// OK
	default:
		return "", 0, fmt.Errorf("invalid status from accrual: %s", status)
	}

	return status, response.Accrual, nil
}
