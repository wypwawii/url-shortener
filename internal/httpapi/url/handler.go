package url

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/wypwawii/url-shortener/internal/config"
)

type URLService interface {
	Save(
		ctx context.Context,
		url string,
		alias *string,
	) (*ShortURL, error)
	Get(ctx context.Context, alias string) (*ShortURL, error)
	Update(ctx context.Context, alias string, newURL string) error
	Delete(ctx context.Context, alias string) error
}

type Handler struct {
	service URLService
	log     *slog.Logger
	auth    config.AuthConfig
}

type SaveRequest struct {
	URL   string  `json:"url" validate:"required,url"`
	Alias *string `json:"alias,omitempty"`
}

type SaveResponse struct {
	Id    int32  `json:"id"`
	Alias string `json:"alias"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HandlerOpts struct {
	Service URLService
	Log     *slog.Logger
}

func NewHandler(opts HandlerOpts) *Handler {
	return &Handler{
		service: opts.Service,
		log:     opts.Log,
	}
}

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var req SaveRequest
	log := h.log.With(
		"operation", "url.Save",
		"request_id", middleware.GetReqID(r.Context()),
	)

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error: "invalid request body",
		})

		return
	}

	result, err := h.service.Save(r.Context(), req.URL, req.Alias)
	if err != nil {
		status := http.StatusInternalServerError
		message := "internal server error"

		var MaxAliasLengthErr *AliasTooLongError
		switch {
		case errors.Is(err, ErrInvalidURL):
			status = http.StatusBadRequest
			message = "invalid URL"
		case errors.Is(err, ErrAliasTooLong):
			status = http.StatusBadRequest
			message = "alias is too long"
		case errors.Is(err, ErrAliasEmpty):
			status = http.StatusBadRequest
			message = "alias must not be empty"
		case errors.Is(err, ErrAliasExists):
			status = http.StatusConflict
			message = "alias already exists"
		case errors.As(err, &MaxAliasLengthErr):
			status = http.StatusBadRequest
			message = MaxAliasLengthErr.Error()
		default:
			log.Error("failed to save URL", "error", err)
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error: message,
		})

		return
	}

	log.Info("URL saved", "url_id", result.Id)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, SaveResponse{
		Id:    result.Id,
		Alias: result.Alias,
	})
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")

	log := h.log.With(
		"operation", "url.Redirect",
		"request_id", middleware.GetReqID(r.Context()),
	)

	result, err := h.service.Get(r.Context(), alias)
	if err != nil {
		status := http.StatusInternalServerError
		message := "internal server error"

		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
			message = "URL not found"
		} else {
			log.Error("failed to resolve alias", "error", err)
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error: message,
		})
		return
	}

	http.Redirect(w, r, result.Value, http.StatusFound)
}

type UpdateRequest struct {
	URL string `json:"url"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")

	log := h.log.With(
		"operation", "url.Update",
		"request_id", middleware.GetReqID(r.Context()),
		"alias", alias,
	)

	var req UpdateRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	if err := h.service.Update(r.Context(), alias, req.URL); err != nil {
		status := http.StatusInternalServerError
		message := "internal server error"

		switch {
		case errors.Is(err, ErrAliasEmpty):
			status = http.StatusBadRequest
			message = "alias must not be empty"

		case errors.Is(err, ErrInvalidURL):
			status = http.StatusBadRequest
			message = "invalid URL"

		case errors.Is(err, ErrNotFound):
			status = http.StatusNotFound
			message = "URL not found"

		default:
			log.Error("failed to update URL", "error", err)
		}

		render.Status(r, status)
		render.JSON(w, r, ErrorResponse{
			Error: message,
		})
		return
	}

	log.Info("URL updated")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")

	log := h.log.With(
		"operation", "url.Delete",
		"request_id", middleware.GetReqID(r.Context()),
		"alias", alias,
	)

	if err := h.service.Delete(r.Context(), alias); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			log.Info("URL already absent")
			w.WriteHeader(http.StatusNoContent)

		case errors.Is(err, ErrAliasEmpty):
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, ErrorResponse{
				Error: "alias must not be empty",
			})

		default:
			log.Error("failed to delete URL", "error", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, ErrorResponse{
				Error: "internal server error",
			})
		}

		return
	}

	log.Info("URL deleted")
	w.WriteHeader(http.StatusNoContent)
}
