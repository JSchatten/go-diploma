package handlers

import (
	"net/http"

	logZero "github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

func Hello() gin.HandlerFunc {
	return func(c *gin.Context) {
		logZero.Logger.Info().Msg("HelloHandler")
		c.JSON(http.StatusOK, gin.H{"result": "ok"})
	}
}
