package server

import (
	"log"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (recorder *responseRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}

	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *responseRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}

	written, err := recorder.ResponseWriter.Write(body)
	recorder.bytes += written
	return written, err
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		log.Printf(
			"http_request method=%s path=%s status=%d bytes=%d duration=%s remote_addr=%s",
			r.Method,
			r.URL.Path,
			status,
			recorder.bytes,
			time.Since(started),
			r.RemoteAddr,
		)
	})
}
