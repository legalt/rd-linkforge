package link

import (
	"errors"
	"net/url"
	"time"

	"github.com/skskuzan/rd-linkforge/internal/base62"
)

var (
	ErrEmptyTarget       = errors.New("link: empty target")
	ErrUnsupportedSchema = errors.New("link: unsupported schema")
)

type Link struct {
	ID        uint64
	Code      string
	Target    string
	CreatedAt time.Time
}

func New(id uint64, target string) (Link, error) {
	if target == "" {
		return Link{}, ErrEmptyTarget
	}

	if _, err := url.ParseRequestURI(target); err != nil {
		return Link{}, ErrUnsupportedSchema
	}

	return Link{
		ID:        id,
		Code:      base62.EncodeWidth(id, 6),
		Target:    target,
		CreatedAt: time.Now(),
	}, nil
}
