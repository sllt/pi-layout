package middleware

import (
	"bytes"
	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/duke-git/lancet/v2/random"
	"github.com/sllt/pi-layout/pkg/log"
	"io"
	"net/http"
	"time"
)

func RequestLogMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uuid, err := random.UUIdV4()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			trace := cryptor.Md5String(uuid)

			var body string
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				body = string(bodyBytes)
			}
			logger.Infof("Request %s %s trace=%s body=%s", r.Method, r.URL.String(), trace, body)
			next.ServeHTTP(w, r)
		})
	}
}

func ResponseLogMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			blw := &bodyLogWriter{
				ResponseWriter: w,
				body:           bytes.NewBufferString(""),
				statusCode:     http.StatusOK,
			}
			startTime := time.Now()
			next.ServeHTTP(blw, r)
			duration := time.Since(startTime).String()
			logger.Infof("Response status=%d time=%s body=%s", blw.statusCode, duration, blw.body.String())
		})
	}
}

type bodyLogWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
