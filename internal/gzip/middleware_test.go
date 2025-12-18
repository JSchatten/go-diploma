package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// compressData сжимает строку с помощью gzip
func compressData(data string) (*bytes.Reader, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(data))
	if err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// decompressData распаковывает gzip-ответ
func decompressData(data []byte) (string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gz.Close()
	out, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func TestGzipMiddleware_CompressesResponse_WhenAccepted(t *testing.T) {
	r := gin.New()
	r.Use(GzipMiddleware())

	r.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, this is a compressible text response!")
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))
	assert.NotEqual(t, "", w.Header().Get("Content-Encoding"))

	// Проверим, что тело сжато и корректно распаковывается
	body, err := decompressData(w.Body.Bytes())
	require.NoError(t, err)
	assert.Equal(t, "Hello, this is a compressible text response!", body)
}

func TestGzipMiddleware_DoesNotCompress_WhenNotAccepted(t *testing.T) {
	r := gin.New()
	r.Use(GzipMiddleware())

	r.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "No compression here")
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	// Нет Accept-Encoding

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Empty(t, w.Header().Get("Vary"))
	assert.Equal(t, "No compression here", w.Body.String())
}

func TestGzipMiddleware_DoesNotCompress_BinaryContentType(t *testing.T) {

	r := gin.New()
	r.Use(GzipMiddleware())

	r.GET("/image", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", []byte{0x89, 0x50, 0x4E, 0x47}) // PNG
		// TODO дописать проверку
		// TODO почеу-то игнорит переназначение Content-Type на выходе
		// c.Header("Content-Type", "image/png")
		// c.Writer.WriteHeader(http.StatusOK)
		// c.Writer.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	})

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	// Нужно ли Accept-Encoding для изображения?
	// req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Empty(t, w.Header().Get("Vary"))
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, w.Body.Bytes())
}

func TestGzipMiddleware_Compresses_JSON(t *testing.T) {
	r := gin.New()
	r.Use(GzipMiddleware())

	r.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello", "value": 42})
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", w.Header().Get("Vary"))

	body, err := decompressData(w.Body.Bytes())
	require.NoError(t, err)

	// Проверим, что JSON валиден
	assert.Contains(t, body, `"message":"hello"`)
	assert.Contains(t, body, `"value":42`)
}

func TestGzipMiddleware_DecompressesRequest_GzipBody(t *testing.T) {
	const originalBody = `{"key": "value"}`
	compressedBody, err := compressData(originalBody)
	require.NoError(t, err)

	r := gin.New()
	r.Use(GzipMiddleware())

	r.POST("/echo", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", compressedBody)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originalBody, w.Body.String())
}

func TestGzipMiddleware_BadGzipRequest_Body(t *testing.T) {
	// Передаём битые данные
	badGzip := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff} // неполные

	r := gin.New()
	r.Use(GzipMiddleware())

	r.POST("/echo", func(c *gin.Context) {
		c.String(http.StatusOK, "processed")
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(badGzip))
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // должен прервать с 400
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		header   string
		expected bool
	}{
		{"gzip", true},
		{"deflate, gzip", true},
		{"gzip;q=1.0, identity;q=0.5", true},
		{"", false},
		{"deflate", false},
		{"compress", false},
		{"gzip; charset=UTF-8", true},
		{"identity", false},
	}

	for _, tt := range tests {
		req := &http.Request{Header: http.Header{}}
		if tt.header != "" {
			req.Header.Set("Accept-Encoding", tt.header)
		}
		assert.Equal(t, tt.expected, acceptsGzip(req), "header: %s", tt.header)
	}
}

func TestShouldCompressContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/html", true},
		{"text/css", true},
		{"application/json", true},
		{"application/javascript", true},
		{"application/xml", true},
		{"", true}, // по умолчанию
		{"image/png", false},
		{"application/octet-stream", false},
		{"application/protobuf", false},
		{"font/woff2", false},
		{"application/pdf", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, shouldCompressContentType(tt.contentType),
			"contentType: %s", tt.contentType)
	}
}
