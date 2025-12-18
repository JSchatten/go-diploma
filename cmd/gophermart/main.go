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

	"github.com/JSchatten/go-diploma/internal/auth"
	"github.com/JSchatten/go-diploma/internal/config"
	gzipMiddleaware "github.com/JSchatten/go-diploma/internal/gzip"
	"github.com/JSchatten/go-diploma/internal/handlers"
	loggingMiddleware "github.com/JSchatten/go-diploma/internal/logging"
	"github.com/JSchatten/go-diploma/internal/storage"

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

	// инициализация хранилища
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, err := storage.NewPSQLStorage(ctx, cfg.DatabaseURI)
	if err != nil {
		logZero.Logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer store.Close()

	err = store.Migarte(ctx)
	if err != nil {
		logZero.Logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	logZero.Logger.Info().Msg("Database connected")

	// gin
	gin.DefaultWriter = io.Discard
	router := gin.New()
	router.RedirectFixedPath = false

	router.Use(loggingMiddleware.LoggingMiddleware(logZero.Logger))
	router.Use(gzipMiddleaware.GzipMiddleware())

	authHandlers := auth.NewAuthHandlers(store)

	// public routes
	router.GET("/", handlers.Hello())
	router.GET("/live", handlers.Hello())
	router.POST("/api/user/register", authHandlers.RegisterHandler)
	router.POST("/api/user/login", authHandlers.LoginHandler)

	// protected routes
	authorized := router.Group("/")
	authorized.Use(authHandlers.AuthMiddleware)
	{
		authorized.GET("/protected", handlers.Hello())
		authorized.POST("/api/user/orders", handlers.Hello())
		authorized.GET("/api/user/orders", handlers.Hello())
		authorized.GET("/api/user/balance", handlers.Hello())
		authorized.POST("/api/user/balance/withdraw", handlers.Hello())
		authorized.GET("/api/user/withdrawals", handlers.Hello())
	}

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

	logZero.Logger.Info().Msg("Server started")

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
