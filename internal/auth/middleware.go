package auth

import (
	"github.com/gin-gonic/gin"
	logZero "github.com/rs/zerolog/log"
)

// var hashSecretHandler string

func AuthMiddleware(hashSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		logZero.Logger.Info().Msg("Use AuthMiddleware")

		c.Next()
	}
}
