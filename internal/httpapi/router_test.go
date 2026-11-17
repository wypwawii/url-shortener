package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wypwawii/url-shortener/internal/config"
	"github.com/wypwawii/url-shortener/internal/httpapi"
	"github.com/wypwawii/url-shortener/internal/httpapi/url"
	"github.com/wypwawii/url-shortener/internal/httpapi/url/mocks"
)

func TestRouterURLLifecycle(t *testing.T) {
	id := int32(42)
	alias := "abc123"
	originalURL := "https://example.com"
	updatedURL := "https://example.org"
	path := "/" + alias

	repo := mocks.NewMockURLRepository(t)

	mock.InOrder(
		repo.EXPECT().
			Save(mock.Anything, originalURL, alias).
			Return(id, nil).
			Once(),

		repo.EXPECT().
			Get(mock.Anything, alias).
			Return(&url.ShortURL{
				Id:    id,
				Value: originalURL,
				Alias: alias,
			}, nil).
			Once(),

		repo.EXPECT().
			Update(mock.Anything, alias, updatedURL).
			Return(nil).
			Once(),

		repo.EXPECT().
			Get(mock.Anything, alias).
			Return(&url.ShortURL{
				Id:    id,
				Value: updatedURL,
				Alias: alias,
			}, nil).
			Once(),

		repo.EXPECT().
			Delete(mock.Anything, alias).
			Return(nil).
			Once(),

		repo.EXPECT().
			Get(mock.Anything, alias).
			Return(nil, url.ErrNotFound).
			Once(),

		repo.EXPECT().
			Delete(mock.Anything, alias).
			Return(url.ErrNotFound).
			Once(),
	)

	service := url.NewService(url.ServiceOpts{
		Repo:           repo,
		MaxAliasLength: 8,
	})

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := url.NewHandler(url.HandlerOpts{
		Service: service,
		Log:     log,
	})

	auth := config.AuthConfig{
		Username: "test-user",
		Password: "test-password",
	}

	router := httpapi.NewRouter(httpapi.RouterOpts{
		Handlers: httpapi.Handlers{
			URLHandler: handler,
		},
		Log:  log,
		Auth: auth,
	})

	saveBody, err := json.Marshal(url.SaveRequest{
		URL:   originalURL,
		Alias: &alias,
	})
	require.NoError(t, err)

	updateBody, err := json.Marshal(url.UpdateRequest{
		URL: updatedURL,
	})
	require.NoError(t, err)

	steps := []struct {
		name         string
		method       string
		path         string
		body         []byte
		wantStatus   int
		wantLocation string
	}{
		{
			name:       "create",
			method:     http.MethodPost,
			path:       "/save",
			body:       saveBody,
			wantStatus: http.StatusCreated,
		},
		{
			name:         "redirect",
			method:       http.MethodGet,
			path:         path,
			wantStatus:   http.StatusFound,
			wantLocation: originalURL,
		},
		{
			name:       "update",
			method:     http.MethodPut,
			path:       path,
			body:       updateBody,
			wantStatus: http.StatusNoContent,
		},
		{
			name:         "redirect after update",
			method:       http.MethodGet,
			path:         path,
			wantStatus:   http.StatusFound,
			wantLocation: updatedURL,
		},
		{
			name:       "delete",
			method:     http.MethodDelete,
			path:       path,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found after delete",
			method:     http.MethodGet,
			path:       path,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete again",
			method:     http.MethodDelete,
			path:       path,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, step := range steps {
		ok := t.Run(step.name, func(t *testing.T) {
			req := httptest.NewRequest(
				step.method,
				step.path,
				bytes.NewReader(step.body),
			)

			if step.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			if step.method != http.MethodGet {
				req.SetBasicAuth(auth.Username, auth.Password)
			}

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(
				t,
				step.wantStatus,
				recorder.Code,
				"response body: %s",
				recorder.Body.String(),
			)

			require.Equal(
				t,
				step.wantLocation,
				recorder.Header().Get("Location"),
			)

			switch step.wantStatus {
			case http.StatusCreated:
				require.Equal(
					t,
					"application/json",
					recorder.Header().Get("Content-Type"),
				)

				var got url.SaveResponse
				err := json.Unmarshal(recorder.Body.Bytes(), &got)
				require.NoError(t, err)

				require.Equal(t, url.SaveResponse{
					Id:    id,
					Alias: alias,
				}, got)

			case http.StatusNoContent:
				require.Empty(t, recorder.Body.String())

			case http.StatusNotFound:
				require.Equal(
					t,
					"application/json",
					recorder.Header().Get("Content-Type"),
				)

				var got url.ErrorResponse
				err := json.Unmarshal(recorder.Body.Bytes(), &got)
				require.NoError(t, err)

				require.Equal(t, url.ErrorResponse{
					Error: "URL not found",
				}, got)
			}
		})

		if !ok {
			break
		}
	}
}
