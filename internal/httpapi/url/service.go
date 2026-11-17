package url

import (
	"context"
	"fmt"

	"github.com/wypwawii/url-shortener/internal/lib"
)

type URLRepository interface {
	Save(ctx context.Context, url string, alias string) (int32, error)
	Get(ctx context.Context, alias string) (*ShortURL, error)
	Update(ctx context.Context, alias string, newURL string) error
	Delete(ctx context.Context, alias string) error
}

type Service struct {
	repo           URLRepository
	MaxAliasLength int
}

type ServiceOpts struct {
	Repo           URLRepository
	MaxAliasLength int
}

func NewService(opts ServiceOpts) *Service {
	return &Service{repo: opts.Repo, MaxAliasLength: opts.MaxAliasLength}
}

func (s *Service) Save(
	ctx context.Context,
	url string,
	alias *string,
) (*ShortURL, error) {
	if !lib.IsValidURL(url) {
		return nil, ErrInvalidURL
	}

	var resolvedAlias string
	var err error

	if alias == nil {
		resolvedAlias, err = RandomString(s.MaxAliasLength)
		if err != nil {
			return nil, fmt.Errorf("generate random alias: %w", err)
		}
	} else {
		resolvedAlias = *alias

		if resolvedAlias == "" {
			return nil, ErrAliasEmpty
		}

		if len(resolvedAlias) > s.MaxAliasLength {
			return nil, &AliasTooLongError{
				MaxLength:    s.MaxAliasLength,
				ActualLength: len(resolvedAlias),
			}
		}
	}

	id, err := s.repo.Save(ctx, url, resolvedAlias)
	if err != nil {
		return nil, fmt.Errorf("create short URL: %w", err)
	}

	return &ShortURL{
		Id:    id,
		Value: url,
		Alias: resolvedAlias,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	alias string,
) (*ShortURL, error) {
	result, err := s.repo.Get(ctx, alias)
	if err != nil {
		return nil, fmt.Errorf("get short URL: %w", err)
	}

	return result, nil
}

func (s *Service) Update(
	ctx context.Context,
	alias string,
	newURL string,
) error {
	if alias == "" {
		return ErrAliasEmpty
	}

	if !lib.IsValidURL(newURL) {
		return ErrInvalidURL
	}

	if err := s.repo.Update(ctx, alias, newURL); err != nil {
		return fmt.Errorf("update short URL: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, alias string) error {
	if alias == "" {
		return ErrAliasEmpty
	}

	if err := s.repo.Delete(ctx, alias); err != nil {
		return fmt.Errorf("delete short URL: %w", err)
	}

	return nil
}
