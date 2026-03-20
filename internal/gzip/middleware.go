package gzip

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		if c.Request.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			defer gz.Close()
			c.Request.Header.Del("Content-Encoding")
			c.Request.Body = gz
		}

		if !acceptsGzip(c.Request) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()
		gw := newGzipWriter(gz, c.Writer)
		c.Writer = gw

		// // Попробую сжимать всё
		// c.Writer.Header().Set("Content-Encoding", "gzip")
		// c.Writer.Header().Set("Vary", "Accept-Encoding")
		// // Удаляем размер, так как он будет неточный из-за сжатия
		// c.Writer.Header().Del("Content-Length")

		contentType := c.Writer.Header().Get("Content-Type")
		if shouldCompressContentType(contentType) {
			c.Writer.Header().Set("Content-Encoding", "gzip")
			c.Writer.Header().Set("Vary", "Accept-Encoding")
			// Удаляем размер, так как он будет неточный из-за сжатия
			c.Writer.Header().Del("Content-Length")
		} else {
			// Удаляем сжатие для бинарных типов, явно
			c.Writer.Header().Del("Content-Encoding")
			c.Writer.Header().Del("Vary")
		}

		c.Next()
	}
}

// acceptsGzip проверяет, что клиент поддерживает gzip
func acceptsGzip(r *http.Request) bool {
	accept := r.Header.Get("Accept-Encoding")
	parts := strings.Split(accept, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return true
		}
	}
	return false
}

var noCompressMIME = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"audio/mpeg":               true,
	"video/mp4":                true,
	"font/woff":                true,
	"font/woff2":               true,
	"application/pdf":          true,
	"application/zip":          true,
	"application/gzip":         true,
	"application/x-tar":        true,
	"application/x-rar":        true,
	"application/octet-stream": true,
	"application/protobuf":     true,
	"application/msgpack":      true,
}

func shouldCompressContentType(contentType string) bool {
	if contentType == "" {
		return true
	}

	// Убираем параметры (например, ;charset=utf-8)
	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = contentType[:i]
	}
	contentType = strings.TrimSpace(contentType)

	if noCompressMIME[strings.ToLower(contentType)] {
		return false
	}

	// Проверяем префиксы
	lower := strings.ToLower(contentType)
	if strings.HasPrefix(lower, "text/") ||
		strings.Contains(lower, "json") ||
		strings.Contains(lower, "xml") ||
		strings.Contains(lower, "javascript") {

		return true
	}

	return false
}
