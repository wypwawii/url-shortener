package url_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wypwawii/url-shortener/internal/httpapi/url"
	"github.com/wypwawii/url-shortener/internal/httpapi/url/mocks"
)

func TestServiceSaveSuccess(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)
	alias := "abcdef"

	repo.EXPECT().
		Save(ctx, "https://example.com", alias).
		Return(int32(42), nil).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo:           repo,
		MaxAliasLength: 8,
	})

	result, err := service.Save(ctx, "https://example.com", &alias)

	require.NoError(t, err)
	require.Equal(t, &url.ShortURL{
		Id:    42,
		Value: "https://example.com",
		Alias: alias,
	}, result)
}

func TestServiceSaveInvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
	}{
		{
			name:     "empty URL",
			inputURL: "",
		},
		{
			name:     "unsupported scheme",
			inputURL: "ftp://example.com",
		},
		{
			name:     "relative path",
			inputURL: "/articles",
		},
		{
			name:     "missing host",
			inputURL: "https://",
		},
		{
			name:     "invalid escape",
			inputURL: "https://example.com/%zz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := mocks.NewMockURLRepository(t)

			service := url.NewService(url.ServiceOpts{
				Repo:           repo,
				MaxAliasLength: 8,
			})

			alias := "abc123"
			result, err := service.Save(ctx, tt.inputURL, &alias)

			require.ErrorIs(t, err, url.ErrInvalidURL)
			require.Nil(t, result)
		})
	}
}

func TestServiceSaveAliasTooLong(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	service := url.NewService(url.ServiceOpts{
		Repo:           repo,
		MaxAliasLength: 8,
	})

	alias := "abcdefghi"
	result, err := service.Save(
		ctx,
		"https://example.com",
		&alias, // 9 characters when 8 is max
	)

	require.Nil(t, result)

	var lengthErr *url.AliasTooLongError
	require.ErrorAs(t, err, &lengthErr)

	require.Equal(t, 8, lengthErr.MaxLength)
	require.Equal(t, 9, lengthErr.ActualLength)
}

func TestServiceSaveAliasExists(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)
	alias := "abcdef"

	repo.EXPECT().
		Save(ctx, "https://example.com", alias).
		Return(int32(0), url.ErrAliasExists).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo:           repo,
		MaxAliasLength: 8,
	})

	result, err := service.Save(
		ctx,
		"https://example.com",
		&alias,
	)

	require.ErrorIs(t, err, url.ErrAliasExists)
	require.Nil(t, result)
}

func TestServiceSaveGeneratesAlias(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	var savedAlias string
	id := int32(42)

	repo.EXPECT().
		Save(
			ctx,
			"https://example.com",
			mock.MatchedBy(func(alias string) bool {
				return len(alias) == 8
			}),
		).
		Run(func(_ context.Context, _ string, alias string) {
			savedAlias = alias
		}).
		Return(int32(id), nil).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo:           repo,
		MaxAliasLength: 8,
	})

	result, err := service.Save(ctx, "https://example.com", nil)

	require.NoError(t, err)
	require.Equal(t, &url.ShortURL{
		Id:    id,
		Value: "https://example.com",
		Alias: savedAlias,
	}, result)
}

func TestServiceGetSuccess(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Get(ctx, "abc123").
		Return(&url.ShortURL{
			Id:    42,
			Value: "https://example.com",
			Alias: "abc123",
		}, nil).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	result, err := service.Get(ctx, "abc123")

	require.NoError(t, err)
	require.Equal(t, &url.ShortURL{
		Id:    42,
		Value: "https://example.com",
		Alias: "abc123",
	}, result)
}

func TestServiceGetNotFound(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Get(ctx, "missing").
		Return(nil, url.ErrNotFound).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	result, err := service.Get(ctx, "missing")

	require.ErrorIs(t, err, url.ErrNotFound)
	require.Nil(t, result)
}

func TestServiceGetRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)
	dbErr := errors.New("database unavailable")

	repo.EXPECT().
		Get(ctx, "abc123").
		Return(nil, dbErr).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	result, err := service.Get(ctx, "abc123")

	require.ErrorIs(t, err, dbErr)
	require.Nil(t, result)
}

func TestServiceUpdateSuccess(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Update(ctx, "abc123", "https://example.org").
		Return(nil).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Update(ctx, "abc123", "https://example.org")

	require.NoError(t, err)
}

func TestServiceUpdateEmptyAlias(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Update(ctx, "", "https://example.org")

	require.ErrorIs(t, err, url.ErrAliasEmpty)
}

func TestServiceUpdateInvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
	}{
		{
			name:     "empty URL",
			inputURL: "",
		},
		{
			name:     "unsupported scheme",
			inputURL: "ftp://example.org",
		},
		{
			name:     "relative path",
			inputURL: "/articles",
		},
		{
			name:     "missing host",
			inputURL: "https://",
		},
		{
			name:     "invalid escape",
			inputURL: "https://example.org/%zz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := mocks.NewMockURLRepository(t)

			// При неверном URL вызов repository не ожидается.
			service := url.NewService(url.ServiceOpts{
				Repo: repo,
			})

			err := service.Update(ctx, "abc123", tt.inputURL)

			require.ErrorIs(t, err, url.ErrInvalidURL)
		})
	}
}

func TestServiceUpdateNotFound(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Update(ctx, "missing", "https://example.org").
		Return(url.ErrNotFound).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Update(ctx, "missing", "https://example.org")

	require.ErrorIs(t, err, url.ErrNotFound)
}

func TestServiceUpdateRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)
	dbErr := errors.New("database unavailable")

	repo.EXPECT().
		Update(ctx, "abc123", "https://example.org").
		Return(dbErr).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Update(ctx, "abc123", "https://example.org")

	require.ErrorIs(t, err, dbErr)
}

func TestServiceDeleteSuccess(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Delete(ctx, "abc123").
		Return(nil).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Delete(ctx, "abc123")

	require.NoError(t, err)
}

func TestServiceDeleteEmptyAlias(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	// Вызов repository не ожидается.
	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Delete(ctx, "")

	require.ErrorIs(t, err, url.ErrAliasEmpty)
}

func TestServiceDeleteNotFound(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)

	repo.EXPECT().
		Delete(ctx, "missing").
		Return(url.ErrNotFound).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Delete(ctx, "missing")

	require.ErrorIs(t, err, url.ErrNotFound)
}

func TestServiceDeleteRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockURLRepository(t)
	dbErr := errors.New("database unavailable")

	repo.EXPECT().
		Delete(ctx, "abc123").
		Return(dbErr).
		Once()

	service := url.NewService(url.ServiceOpts{
		Repo: repo,
	})

	err := service.Delete(ctx, "abc123")

	require.ErrorIs(t, err, dbErr)
}
