package link

import (
	"fmt"

	"github.com/skskuzan/rd-linkforge/internal/base62"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

func (p Page) Normalize() Page {
	switch {
	case p.Limit <= 0:
		p.Limit = DefaultPageLimit
	case p.Limit > MaxPageLimit:
		p.Limit = MaxPageLimit
	}

	return p
}

func (p Page) AfterID() (uint64, error) {
	if p.Cursor == "" {
		return 0, nil
	}

	id, err := base62.Decode(p.Cursor)
	if err != nil {
		return 0, fmt.Errorf("decode cursor: %w", ErrInvalidCursor)
	}

	return id, nil
}
