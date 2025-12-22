package handlers

import (
	"net/http"

	"github.com/JSchatten/go-diploma/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func GetBalanceHandler(store storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Warn().Msg("User not authenticated")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		current, withdrawn, err := store.GetBalance(c.Request.Context(), userID.(int64))
		if err != nil {
			log.Error().Err(err).Msg("Failed to get balance")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		log.Debug().Float64("current", current).Float64("withdrawn", withdrawn).Msg("Balance retrieved")
		c.JSON(http.StatusOK, gin.H{
			"current":   current,
			"withdrawn": withdrawn,
		})
	}
}
