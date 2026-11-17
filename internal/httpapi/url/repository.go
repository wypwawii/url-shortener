package url

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(ctx context.Context, db *pgxpool.Pool) (*Repository, error) {
	_, err := db.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS stored_urls (
            id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
            url TEXT NOT NULL,
            alias TEXT NOT NULL UNIQUE
        )
    `)

	if err != nil {
		return nil, fmt.Errorf("create stored_urls table: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Save(ctx context.Context, url string, alias string) (int32, error) {
	const query = `
        INSERT INTO stored_urls (url, alias)
        VALUES ($1, $2)
        RETURNING id
    `

	var id int32

	err := r.db.QueryRow(ctx, query, url, alias).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "stored_urls_alias_key" {
			return 0, ErrAliasExists
		}

		return 0, fmt.Errorf("save URL: %w", err)
	}

	return id, nil
}

func (r *Repository) Get(ctx context.Context, alias string) (*ShortURL, error) {
	const query = `
        SELECT id, url, alias
        FROM stored_urls
        WHERE alias = $1
    `

	var result ShortURL

	err := r.db.QueryRow(ctx, query, alias).
		Scan(&result.Id, &result.Value, &result.Alias)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get URL by alias: %w", err)
	}

	return &result, nil
}

func (r *Repository) Delete(ctx context.Context, alias string) error {
	const query = `DELETE FROM stored_urls WHERE alias = $1`

	result, err := r.db.Exec(ctx, query, alias)
	if err != nil {
		return fmt.Errorf("delete URL: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) Update(
	ctx context.Context,
	alias string,
	newURL string,
) error {
	const query = `
		UPDATE stored_urls
		SET url = $1
		WHERE alias = $2
	`

	result, err := r.db.Exec(ctx, query, newURL, alias)
	if err != nil {
		return fmt.Errorf("update URL: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
