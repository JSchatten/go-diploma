// internal/handlers/withdraw.go
package handlers

import (
	"errors"
	"net/http"

	"github.com/JSchatten/go-diploma/internal/models"
	"github.com/JSchatten/go-diploma/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func WithdrawHandler(balanceService *service.BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var req models.WithdrawRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Warn().Err(err).Msg("Invalid withdrawal request")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		err := balanceService.Withdraw(c.Request.Context(), userID.(int64), req.Order, req.Sum)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidSum), errors.Is(err, service.ErrInvalidOrder):
				log.Warn().Err(err).Msg("Invalid withdrawal data")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			case errors.Is(err, service.ErrInsufficientFunds):
				log.Warn().Int64("user_id", userID.(int64)).Float64("sum", req.Sum).Msg("Insufficient funds")
				c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient funds"})
			default:
				log.Error().Err(err).Msg("Failed to withdraw")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			}
			return
		}

		log.Info().Int64("user_id", userID.(int64)).Float64("sum", req.Sum).Str("order", req.Order).Msg("Withdrawal successful")
		c.Status(http.StatusOK)
	}
}

func GetWithdrawalsHandler(balanceService *service.BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		withdrawals, err := balanceService.GetWithdrawals(c.Request.Context(), userID.(int64))
		if err != nil {
			log.Error().Err(err).Msg("Failed to load withdrawals")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		if len(withdrawals) == 0 {
			log.Debug().Int64("user_id", userID.(int64)).Msg("No withdrawals found")
			c.Header("Content-Type", "application/json")
			c.Status(http.StatusNoContent)
			return
		}

		log.Info().Int("count", len(withdrawals)).Msg("Withdrawals retrieved")
		c.JSON(http.StatusOK, withdrawals)
	}
}
