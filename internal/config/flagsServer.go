package config

import (
	"flag"
	"fmt"
	"os"
)

// ServerFlags хранит конфигурацию запуска сервера
type ServerFlags struct {
	RunAddress        string // Адрес и порт запуска HTTP-сервера
	DatabaseURI       string // DSN для подключения к PostgreSQL
	AccrualSystemAddr string // Адрес внешней системы расчёта начислений
}

const (
	defaultRunAddress       = "localhost:8080"
	defaultAccrualSystemURL = "http://localhost:8081"
)

// InitServerFlags инициализирует флаги и переменные окружения
func InitServerFlags() (*ServerFlags, error) {
	var (
		runAddr           = new(string)
		databaseURI       = new(string)
		accrualSystemAddr = new(string)
	)

	// Установим значения по умолчанию
	*runAddr = defaultRunAddress
	*accrualSystemAddr = defaultAccrualSystemURL

	// Читаем переменные окружения (имеют приоритет)
	if v, exists := os.LookupEnv("RUN_ADDRESS"); exists {
		*runAddr = v
	}
	if v, exists := os.LookupEnv("DATABASE_URI"); exists {
		*databaseURI = v
	}
	if v, exists := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); exists {
		*accrualSystemAddr = v
	}

	// Определяем флаги
	flag.StringVar(runAddr, "a", *runAddr, fmt.Sprintf("Server address and port (default: %s)", defaultRunAddress))
	flag.StringVar(databaseURI, "d", *databaseURI, "Database connection URI")
	flag.StringVar(accrualSystemAddr, "r", *accrualSystemAddr, fmt.Sprintf("Accrual system address (default: %s)", defaultAccrualSystemURL))

	// Парсим флаги
	flag.Parse()

	// Проверяем, были ли флаги заданы явно (через command-line)
	wasDatabaseFlagSet := false
	wasAccrualFlagSet := false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "d":
			wasDatabaseFlagSet = true
		case "r":
			wasAccrualFlagSet = true
		}
	})

	if wasDatabaseFlagSet && *databaseURI == "" {
		return nil, fmt.Errorf("flag -d (DATABASE_URI) requires a non-empty value")
	}

	if wasAccrualFlagSet && *accrualSystemAddr == "" {
		return nil, fmt.Errorf("flag -r (ACCRUAL_SYSTEM_ADDRESS) requires a non-empty value")
	}

	if os.Getenv("DATABASE_URI") != "" && *databaseURI == "" {
		return nil, fmt.Errorf("environment variable DATABASE_URI is set but empty")
	}

	if os.Getenv("ACCRUAL_SYSTEM_ADDRESS") != "" && *accrualSystemAddr == "" {
		return nil, fmt.Errorf("environment variable ACCRUAL_SYSTEM_ADDRESS is set but empty")
	}

	// Собираем результат
	return &ServerFlags{
		RunAddress:        *runAddr,
		DatabaseURI:       *databaseURI,
		AccrualSystemAddr: *accrualSystemAddr,
	}, nil
}
