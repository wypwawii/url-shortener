package url

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidURL   = errors.New("invalid URL")
	ErrAliasTooLong = errors.New("alias is too long")
	ErrAliasExists  = errors.New("alias already exists")
	ErrNotFound     = errors.New("not found")
	ErrAliasEmpty   = errors.New("alias must not be empty")
)

type AliasTooLongError struct {
	MaxLength    int
	ActualLength int
}

func (e *AliasTooLongError) Error() string {
	return fmt.Sprintf(
		"alias must be at most %d characters, got %d",
		e.MaxLength,
		e.ActualLength,
	)
}
