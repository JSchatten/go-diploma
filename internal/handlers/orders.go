// internal/handlers/orders.go
package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/JSchatten/go-diploma/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func AddOrderHandler(orderService *service.OrderService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Warn().Err(err).Msg("Cannot read request body")
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "cannot read request body"})
			return
		}
		number := strings.TrimSpace(string(body))

		err = orderService.UploadOrder(c.Request.Context(), userID.(int64), number)
		if err != nil {
			switch {
			case err.Error() == "empty order number":
				log.Warn().Msg("Empty order number")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "empty order number"})
			case errors.Is(err, service.ErrInvalidOrderFormat):
				log.Debug().Str("number", number).Msg("Luhn check failed")
				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid order format"})
			case errors.Is(err, service.ErrOrderBelongsToUser):
				log.Debug().Str("number", number).Msg("Order already uploaded by user")
				c.AbortWithStatusJSON(http.StatusOK, nil)
			case errors.Is(err, service.ErrOrderExists):
				log.Warn().Str("number", number).Msg("Order belongs to another user")
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "order belongs to another user"})
			default:
				log.Error().Err(err).Msg("Failed to save order")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save order"})
			}
			return
		}

		log.Info().Int64("user_id", userID.(int64)).Str("order", number).Msg("Order uploaded")
		c.Status(http.StatusAccepted)
	}
}

func GetOrdersHandler(orderService *service.OrderService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		orders, err := orderService.GetOrders(c.Request.Context(), userID.(int64))
		if err != nil {
			log.Error().Err(err).Msg("Failed to load orders")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		if len(orders) == 0 {
			log.Debug().Int64("user_id", userID.(int64)).Msg("No orders found")
			c.Header("Content-Type", "application/json")
			c.Status(http.StatusNoContent)
			return
		}

		log.Info().Int("count", len(orders)).Msg("Orders retrieved")
		c.JSON(http.StatusOK, orders)
	}
}
