package logging

import "github.com/gin-gonic/gin"

type responseWriter struct {
	gin.ResponseWriter
	status   int
	bodySize int
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.bodySize += size
	return size, err
}

func (rw *responseWriter) BodySize() int {
	return rw.bodySize
}
