package url_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/wypwawii/url-shortener/internal/httpapi/url"
	"github.com/wypwawii/url-shortener/internal/httpapi/url/mocks"
)

func TestHandlerSaveSuccess(t *testing.T) {
	alias := "abc123"
	payload := url.SaveRequest{
		URL:   "https://example.com",
		Alias: &alias,
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Save(req.Context(), payload.URL, payload.Alias).
		Return(&url.ShortURL{
			Id:    42,
			Value: payload.URL,
			Alias: alias,
		}, nil).
		Once()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.SaveResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.SaveResponse{
		Id:    42,
		Alias: alias,
	}, got)
}

func TestHandlerSaveAliasExists(t *testing.T) {
	alias := "abc123"
	payload := url.SaveRequest{
		URL:   "https://example.com",
		Alias: &alias,
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Save(req.Context(), payload.URL, payload.Alias).
		Return(nil, url.ErrAliasExists).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "alias already exists",
	}, got)
}

func TestHandlerSaveInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		strings.NewReader(`{"url":`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "invalid request body",
	}, got)
}

func TestHandlerSaveInvalidData(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		alias       string
		serviceErr  error
		wantMessage string
	}{
		{
			name:        "empty URL",
			inputURL:    "",
			alias:       "abc123",
			serviceErr:  url.ErrInvalidURL,
			wantMessage: "invalid URL",
		},
		{
			name:        "unsupported URL scheme",
			inputURL:    "ftp://example.com",
			alias:       "abc123",
			serviceErr:  url.ErrInvalidURL,
			wantMessage: "invalid URL",
		},
		{
			name:     "alias too long",
			inputURL: "https://example.com",
			alias:    "abcdefghi",
			serviceErr: &url.AliasTooLongError{
				MaxLength:    8,
				ActualLength: 9,
			},
			wantMessage: "alias must be at most 8 characters, got 9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := url.SaveRequest{
				URL:   tt.inputURL,
				Alias: &tt.alias,
			}

			body, err := json.Marshal(payload)
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPost,
				"/save",
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			service := mocks.NewMockURLService(t)
			service.EXPECT().
				Save(req.Context(), payload.URL, payload.Alias).
				Return(nil, tt.serviceErr).
				Once()

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := url.NewHandler(url.HandlerOpts{
				Service: service,
				Log:     log,
			})

			handler.Save(recorder, req)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(
				t,
				"application/json",
				recorder.Header().Get("Content-Type"),
			)

			var got url.ErrorResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &got)
			require.NoError(t, err)

			require.Equal(t, url.ErrorResponse{
				Error: tt.wantMessage,
			}, got)
		})
	}
}

func TestHandlerSaveMissingAlias(t *testing.T) {
	payload := url.SaveRequest{
		URL: "https://example.com",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Save(req.Context(), payload.URL, payload.Alias).
		Return(&url.ShortURL{
			Id:    42,
			Value: payload.URL,
			Alias: "aB3dE7fG",
		}, nil).
		Once()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.SaveResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.SaveResponse{
		Id:    42,
		Alias: "aB3dE7fG",
	}, got)
}

func TestHandlerSaveEmptyAlias(t *testing.T) {
	var alias *string = nil
	payload := url.SaveRequest{
		URL:   "https://example.com",
		Alias: alias,
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Save(req.Context(), payload.URL, payload.Alias).
		Return(nil, url.ErrAliasEmpty).
		Once()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "alias must not be empty",
	}, got)
}

func TestHandlerSaveInternalError(t *testing.T) {
	alias := "abc123"
	payload := url.SaveRequest{
		URL:   "https://example.com",
		Alias: &alias,
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/save",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Save(req.Context(), payload.URL, payload.Alias).
		Return(nil, errors.New("database unavailable")).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Save(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "internal server error",
	}, got)
}

func TestHandlerRedirectSuccess(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/abc123",
		nil,
	)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")

	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Get(req.Context(), "abc123").
		Return(&url.ShortURL{
			Id:    42,
			Value: "https://example.com",
			Alias: "abc123",
		}, nil).
		Once()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Redirect(recorder, req)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(
		t,
		"https://example.com",
		recorder.Header().Get("Location"),
	)
}

func TestHandlerRedirectNotFound(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/missing",
		nil,
	)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "missing")

	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Get(req.Context(), "missing").
		Return(nil, url.ErrNotFound).
		Once()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	handler.Redirect(recorder, req)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)
	require.Empty(t, recorder.Header().Get("Location"))

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "URL not found",
	}, got)
}

func TestHandlerRedirectInternalError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Get(req.Context(), "abc123").
		Return(nil, errors.New("database unavailable")).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Redirect(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)
	require.Empty(t, recorder.Header().Get("Location"))

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "internal server error",
	}, got)
}

func TestHandlerUpdateSuccess(t *testing.T) {
	payload := url.UpdateRequest{
		URL: "https://example.org",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPut,
		"/abc123",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Update(req.Context(), "abc123", payload.URL).
		Return(nil).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Update(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
}

func TestHandlerUpdateInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPut,
		"/abc123",
		strings.NewReader(`{"url":`),
	)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	// При сломанном JSON сервис не должен вызываться.
	service := mocks.NewMockURLService(t)

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Update(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "invalid request body",
	}, got)
}

func TestHandlerUpdateInvalidData(t *testing.T) {
	tests := []struct {
		name        string
		alias       string
		inputURL    string
		serviceErr  error
		wantMessage string
	}{
		{
			name:        "empty alias",
			alias:       "",
			inputURL:    "https://example.org",
			serviceErr:  url.ErrAliasEmpty,
			wantMessage: "alias must not be empty",
		},
		{
			name:        "empty URL",
			alias:       "abc123",
			inputURL:    "",
			serviceErr:  url.ErrInvalidURL,
			wantMessage: "invalid URL",
		},
		{
			name:        "unsupported URL scheme",
			alias:       "abc123",
			inputURL:    "ftp://example.org",
			serviceErr:  url.ErrInvalidURL,
			wantMessage: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := url.UpdateRequest{
				URL: tt.inputURL,
			}

			body, err := json.Marshal(payload)
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodPut,
				"/"+tt.alias,
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("alias", tt.alias)
			req = req.WithContext(
				context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
			)

			recorder := httptest.NewRecorder()

			service := mocks.NewMockURLService(t)
			service.EXPECT().
				Update(req.Context(), tt.alias, payload.URL).
				Return(tt.serviceErr).
				Once()

			handler := url.NewHandler(url.HandlerOpts{
				Service: service,
				Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			handler.Update(recorder, req)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(
				t,
				"application/json",
				recorder.Header().Get("Content-Type"),
			)

			var got url.ErrorResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &got)
			require.NoError(t, err)

			require.Equal(t, url.ErrorResponse{
				Error: tt.wantMessage,
			}, got)
		})
	}
}

func TestHandlerUpdateNotFound(t *testing.T) {
	payload := url.UpdateRequest{
		URL: "https://example.org",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPut,
		"/missing",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "missing")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Update(req.Context(), "missing", payload.URL).
		Return(url.ErrNotFound).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Update(recorder, req)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "URL not found",
	}, got)
}

func TestHandlerUpdateInternalError(t *testing.T) {
	payload := url.UpdateRequest{
		URL: "https://example.org",
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPut,
		"/abc123",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Update(req.Context(), "abc123", payload.URL).
		Return(errors.New("database unavailable")).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Update(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "internal server error",
	}, got)
}

func TestHandlerDeleteSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/abc123", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Delete(req.Context(), "abc123").
		Return(nil).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Delete(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
}

func TestHandlerDeleteNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/missing", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "missing")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Delete(req.Context(), "missing").
		Return(fmt.Errorf("delete short URL: %w", url.ErrNotFound)).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Delete(recorder, req)

	// Отсутствующий alias тоже считается успешным результатом удаления.
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
}

func TestHandlerDeleteEmptyAlias(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Delete(req.Context(), "").
		Return(url.ErrAliasEmpty).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Delete(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "alias must not be empty",
	}, got)
}

func TestHandlerDeleteInternalError(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/abc123", nil)

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("alias", "abc123")
	req = req.WithContext(
		context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx),
	)

	recorder := httptest.NewRecorder()

	service := mocks.NewMockURLService(t)
	service.EXPECT().
		Delete(req.Context(), "abc123").
		Return(fmt.Errorf("delete short URL: database unavailable")).
		Once()

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	handler.Delete(recorder, req)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(
		t,
		"application/json",
		recorder.Header().Get("Content-Type"),
	)

	var got url.ErrorResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err)

	require.Equal(t, url.ErrorResponse{
		Error: "internal server error",
	}, got)
}
