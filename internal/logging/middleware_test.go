package logging

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
	// Перехватываем вывод логгера
	logOutput := &bytes.Buffer{}
	logger := zerolog.New(logOutput).Level(zerolog.InfoLevel)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LoggingMiddleware(logger))

	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, world!", w.Body.String())

	logContent := logOutput.String()
	require.NotEmpty(t, logContent, "log is empty")
}

func TestLoggingMiddleware_StatusCapture(t *testing.T) {
	logOutput := &bytes.Buffer{}
	logger := zerolog.New(logOutput).Level(zerolog.InfoLevel)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LoggingMiddleware(logger))

	r.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "Something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	logContent := logOutput.String()
	assert.Contains(t, logContent, `"status":500`)
	assert.Contains(t, logContent, `"body_size":20`)
}

func TestLoggingMiddleware_PostRequest(t *testing.T) {
	logOutput := &bytes.Buffer{}
	logger := zerolog.New(logOutput).Level(zerolog.InfoLevel)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LoggingMiddleware(logger))

	r.POST("/submit", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": 42})
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	logContent := logOutput.String()
	assert.Contains(t, logContent, `"method":"POST"`)
	assert.Contains(t, logContent, `"uri":"/submit"`)
	assert.Contains(t, logContent, `"status":201`)
	// {"id":42} → 9 байт
	assert.Contains(t, logContent, `"body_size":9`)
}

func TestLoggingMiddleware_EmptyResponse(t *testing.T) {
	logOutput := &bytes.Buffer{}
	logger := zerolog.New(logOutput).Level(zerolog.InfoLevel)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LoggingMiddleware(logger))

	r.GET("/empty", func(c *gin.Context) {
		c.Status(http.StatusNoContent) // без тела
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	logContent := logOutput.String()
	assert.Contains(t, logContent, `"status":204`)
	assert.Contains(t, logContent, `"body_size":0`)
}

func TestLoggingMiddleware_Redirect(t *testing.T) {
	logOutput := &bytes.Buffer{}
	logger := zerolog.New(logOutput).Level(zerolog.InfoLevel)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LoggingMiddleware(logger))

	r.GET("/redirect", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/new")
	})

	req := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)

	logContent := logOutput.String()
	assert.Contains(t, logContent, `"status":301`)
	assert.Contains(t, logContent, `"body_size":`)
	// проверим только наличие числа, т.к. размер может меняться
	assert.Regexp(t, `"body_size":\d+`, logContent)
}
