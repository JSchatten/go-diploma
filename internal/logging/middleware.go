package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func LoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Обёртываем Writer, чтобы отследить статус и размер
		writer := &responseWriter{bodySize: 0, ResponseWriter: c.Writer}

		c.Writer = writer

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		logger.Info().
			Str("method", c.Request.Method).
			Str("uri", c.Request.RequestURI).
			Int("status", writer.Status()).
			Int("body_size", writer.BodySize()).
			Dur("duration", duration).
			Msg("handled func")
	}
}
