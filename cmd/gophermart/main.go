package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log"

	"github.com/JSchatten/go-diploma/internal/config"
	gzipMiddleaware "github.com/JSchatten/go-diploma/internal/gzip"
	"github.com/JSchatten/go-diploma/internal/handlers"
	loggingMiddleware "github.com/JSchatten/go-diploma/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	logZero "github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logZero.Logger = logZero.Output(zerolog.ConsoleWriter{Out: log.Writer()})

	cfg, err := config.InitServerFlags()
	if err != nil {
		logZero.Logger.Fatal().Err(err).Msg("Failed to init Server Flags")
	}

	// gin
	gin.DefaultWriter = io.Discard
	router := gin.New()
	router.RedirectFixedPath = false

	router.Use(loggingMiddleware.LoggingMiddleware(logZero.Logger))
	router.Use(gzipMiddleaware.GzipMiddleware())

	// public routes
	router.GET("/", handlers.Hello())
	router.GET("/live", handlers.Hello())
	router.GET("/protected", handlers.Hello())
	router.POST("/api/user/register", handlers.Hello())
	router.POST("/api/user/login", handlers.Hello())

	// protected routes
	router.POST("/api/user/orders", handlers.Hello())
	router.GET("/api/user/orders", handlers.Hello())
	router.GET("/api/user/balance", handlers.Hello())
	router.POST("/api/user/balance/withdraw", handlers.Hello())
	router.GET("/api/user/withdrawals", handlers.Hello())

	// Запуск сервера в отдельной горутине
	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	go func() {
		logZero.Logger.Info().Msgf("Server starting at %s", cfg.RunAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logZero.Logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Перехват сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logZero.Logger.Info().Msg("Shutting down server...")

	// Контекст для graceful shutdown
	ctxShut, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShut()

	// Останавливаем сервер
	if err := srv.Shutdown(ctxShut); err != nil {
		logZero.Logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	logZero.Logger.Info().Msg("Server exited gracefully")

}
