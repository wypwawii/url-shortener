package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func Recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				value := recover()
				if value == nil {
					return
				}

				if value == http.ErrAbortHandler {
					panic(value)
				}

				log.ErrorContext(
					r.Context(),
					"panic while handling request",
					"request_id", middleware.GetReqID(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic", fmt.Sprint(value),
					"stack", string(debug.Stack()),
				)

				if ww.Status() != 0 {
					panic(http.ErrAbortHandler)
				}

				render.Status(r, http.StatusInternalServerError)
				render.JSON(ww, r, map[string]string{
					"error": "internal server error",
				})
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
