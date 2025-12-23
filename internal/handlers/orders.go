package handlers

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/JSchatten/go-diploma/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AddOrderHandler загружает номер заказа для расчёта
func AddOrderHandler(store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {

		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		log.Debug().Msgf("body = %s", body)

		if err != nil {
			log.Warn().Err(err).Msg("Cannot read request body")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "cannot read request body"})
			return
		}
		number := strings.TrimSpace(string(body))

		if number == "" {
			log.Warn().Msg("Empty order number")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "empty order number"})
			return
		}

		if !utils.LuhnCheck(number) {
			log.Debug().Str("number", number).Msg("Luhn check failed")
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid order format"})
			return
		}

		existing, err := store.GetOrder(c.Request.Context(), number)
		if err != nil && err != storage.ErrOrderNotFound {
			log.Debug().Err(err).Msgf("err type: %T\n", err)
			log.Error().Err(err).Msg("Failed to check order existence")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		if existing != nil {
			if existing.UserID == userID.(int64) {
				log.Debug().Str("number", number).Msg("Order already uploaded by user")
				c.AbortWithStatusJSON(http.StatusOK, gin.H{"error": "StatusOK"})
				return
			} else {
				log.Warn().Str("number", number).Msg("Order belongs to another user")
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "order belongs to another user"})
				return
			}
		}

		op := &models.BalanceOperation{
			UserID:        userID.(int64),
			OrderNumber:   number,
			Amount:        0,
			OperationType: models.AccrualOp,
			Status:        models.NewStatus,
			ProcessedAt:   time.Now(),
		}

		if err := store.CreateOperation(c.Request.Context(), op); err != nil {
			log.Error().Err(err).Msg("Failed to save order")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save order"})
			return
		}

		log.Info().Int64("user_id", userID.(int64)).Str("order", number).Msg("Order uploaded")
		c.AbortWithStatusJSON(http.StatusAccepted, gin.H{"error": "Order uploaded"}) // 202
	}
}

func GetOrdersHandler(store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		useCtxrID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userID := useCtxrID.(int64)

		ops, err := store.GetAccrualsByUser(c.Request.Context(), userID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to load orders")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		if len(ops) == 0 {
			log.Debug().Int64("user_id", userID).Msg("No orders found")
			c.AbortWithStatusJSON(http.StatusNoContent, gin.H{"error": "No orders found"})
			return
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

		log.Info().Int("count", len(result)).Msg("Orders retrieved")
		c.JSON(http.StatusOK, result)
	}
}
