package logger

import (
	"log"
	"net/http"
	"os"
	"time"
)

// Logger middleware.
type Logger struct {
	h   http.Handler
	log *log.Logger
}

// SetLogger sets the logger to `log`. If you have used logger.New(), you can use this to set your
// logger. Alternatively, if you already have your log.Logger, then you can just call logger.NewLogger() directly.
func (l *Logger) SetLogger(log *log.Logger) {
	l.log = log
}

// wrapper to capture status.
type wrapper struct {
	http.ResponseWriter
	written int
	status  int
}

// capture status.
func (w *wrapper) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// capture written bytes.
func (w *wrapper) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += n
	return n, err
}

// New logger middleware.
func New() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &Logger{
			log: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC),
			h:   h,
		}
	}
}

// NewLogger logger middleware with the given logger.
func NewLogger(log *log.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return &Logger{
			log: log,
			h:   h,
		}
	}
}

// ServeHTTP implementation.
func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	res := &wrapper{w, 0, 200}

	// output the initial log line
	l.log.Printf("--- %s %s\n", r.Method, r.RequestURI)

	// continue to the next middleware
	l.h.ServeHTTP(res, r)

	// output the final log line
	l.log.Printf("%d %s %s %d %s\n", res.status, r.Method, r.RequestURI, res.written, time.Since(start))
}
