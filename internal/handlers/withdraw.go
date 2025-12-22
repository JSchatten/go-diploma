package handlers

import (
	"net/http"
	"time"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/JSchatten/go-diploma/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func WithdrawHandler(store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userId := userID.(int64)

		var req models.WithdrawRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Warn().Err(err).Msg("Invalid withdrawal request")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if !utils.LuhnCheck(req.Order) {
			log.Debug().Str("order", req.Order).Msg("Invalid order in withdrawal")
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid order number"})
			return
		}

		if req.Sum <= 0 {
			log.Warn().Float64("sum", req.Sum).Msg("Withdrawal sum must be positive")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Sum must be positive"})
			return
		}

		op := &models.BalanceOperation{
			UserID:        userId,
			OrderNumber:   req.Order,
			Amount:        -req.Sum,
			OperationType: models.WithdrawalOp,
			Status:        models.ProcessedStatus,
			ProcessedAt:   time.Now(),
		}

		if err := store.CreateOperation(c.Request.Context(), op); err != nil {
			if err == storage.ErrNoMoney {
				log.Warn().Int64("user_id", userId).Float64("sum", req.Sum).Msg("Insufficient funds")
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient funds"})
				return
			}
			log.Error().Err(err).Msg("Failed to withdraw")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		log.Info().Int64("user_id", userId).Float64("sum", req.Sum).Str("order", req.Order).Msg("Withdrawal successful")
		c.AbortWithStatusJSON(http.StatusOK, gin.H{"error": "Withdrawal successful"})
	}
}

func GetWithdrawalsHandler(store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userId := userID.(int64)

		ops, err := store.GetWithdrawalsByUser(c.Request.Context(), userId)
		if err != nil {
			log.Error().Err(err).Msg("Failed to load withdrawals")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		if len(ops) == 0 {
			log.Debug().Int64("user_id", userId).Msg("No withdrawals found")
			c.AbortWithStatusJSON(http.StatusNoContent, gin.H{"error": "No withdrawals found"})
			return
		}

		var result []models.WithdrawalResponse
		for _, op := range ops {
			result = append(result, models.WithdrawalResponse{
				Order:       op.OrderNumber,
				Sum:         op.Sum,
				ProcessedAt: op.ProcessedAt,
			})
		}

		log.Info().Int("count", len(result)).Msg("Withdrawals retrieved")
		c.JSON(http.StatusOK, result)
	}
}
