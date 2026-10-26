package link

import (
	"errors"
	"net/url"
	"time"

	"github.com/skskuzan/rd-linkforge/internal/base62"
)

var (
	ErrEmptyTarget       = errors.New("link: empty target")
	ErrUnsupportedScheme = errors.New("link: unsupported schema")
	ErrNotFound          = errors.New("link: not found")
	ErrAliasTaken        = errors.New("link: alias already taken")
	ErrInvalidAlias      = errors.New("link: invalid alias")
	ErrReservedAlias     = errors.New("link: reserved alias")
	ErrForbidden         = errors.New("link: forbidden")
	ErrExpired           = errors.New("link: expired")
	ErrInvalidExpiry     = errors.New("link: invalid expiry")
	ErrEmptyOwner        = errors.New("link: owner is required")
	ErrInvalidCursor     = errors.New("link: invalid cursor")
)

func New(id uint64, target string) (Link, error) {
	if target == "" {
		return Link{}, ErrEmptyTarget
	}

	u, err := url.ParseRequestURI(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Link{}, ErrUnsupportedScheme
	}

	return Link{
		ID:        id,
		Code:      base62.EncodeWidth(id, 6),
		Target:    target,
		CreatedAt: time.Now(),
	}, nil
}
