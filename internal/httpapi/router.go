package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wypwawii/url-shortener/internal/config"
	mw "github.com/wypwawii/url-shortener/internal/httpapi/middleware"
	"github.com/wypwawii/url-shortener/internal/httpapi/url"
)

type Handlers struct {
	URLHandler *url.Handler
}

type RouterOpts struct {
	Handlers Handlers
	Log      *slog.Logger
	Auth     config.AuthConfig
}

func NewRouter(opts RouterOpts) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(mw.RequestLogger(opts.Log))
	r.Use(mw.Recoverer(opts.Log))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	username, password := opts.Auth.Username, opts.Auth.Password

	auth := middleware.BasicAuth(
		"url-shortener",
		map[string]string{
			username: password,
		},
	)

	r.With(auth).Post("/save", opts.Handlers.URLHandler.Save)
	r.With(auth).Put("/{alias}", opts.Handlers.URLHandler.Update)
	r.With(auth).Delete("/{alias}", opts.Handlers.URLHandler.Delete)
	r.Get("/{alias}", opts.Handlers.URLHandler.Redirect)

	return r
}
