package link

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/skskuzan/rd-linkforge/internal/base62"
)

const (
	generatedCodeWidth = 6
	maxCreateAttempts  = 3
)

// Service implements Shortener. The ID counter lives on the pointer, so a
// value receiver would hand out the same ID forever.
type Service struct {
	store  Store
	now    func() time.Time
	nextID uint64
}

var _ Shortener = (*Service)(nil)

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

func NewService(store Store, opts ...Option) *Service {
	s := &Service{
		store:  store,
		now:    time.Now,
		nextID: 1,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Service) Shorten(ctx context.Context, req ShortenRequest) (Link, error) {
	if req.OwnerID == "" {
		return Link{}, ErrEmptyOwner
	}

	if _, err := New(s.nextID, req.Target); err != nil {
		return Link{}, err
	}

	now := s.now()
	if !req.ExpiresAt.IsZero() && !req.ExpiresAt.After(now) {
		return Link{}, ErrInvalidExpiry
	}

	if req.Alias != "" {
		if err := checkAlias(req.Alias); err != nil {
			return Link{}, err
		}
	}

	attempts := 1
	if req.Alias == "" {
		attempts = maxCreateAttempts
	}

	for range attempts {
		code := req.Alias
		if code == "" {
			code = base62.EncodeWidth(s.nextID, generatedCodeWidth)
		}

		created, err := s.store.Create(ctx, Link{
			ID:        s.nextID,
			Code:      code,
			Target:    req.Target,
			OwnerID:   req.OwnerID,
			CreatedAt: now.UTC(),
			ExpiresAt: req.ExpiresAt,
		})
		if err == nil {
			s.nextID++
			return created, nil
		}
		if !errors.Is(err, ErrAliasTaken) {
			return Link{}, fmt.Errorf("create link: %w", err)
		}
		if req.Alias != "" {
			return Link{}, &AliasError{Alias: req.Alias, Err: ErrAliasTaken}
		}

		s.nextID++
	}

	return Link{}, fmt.Errorf("create link: %w", ErrAliasTaken)
}

func (s *Service) Resolve(ctx context.Context, code string) (Link, error) {
	l, err := s.store.ByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Link{}, ErrNotFound
		}

		return Link{}, fmt.Errorf("resolve link: %w", err)
	}

	if !l.ExpiresAt.IsZero() && !s.now().Before(l.ExpiresAt) {
		return Link{}, ErrExpired
	}

	return l, nil
}

func (s *Service) List(ctx context.Context, ownerID string, page Page) ([]Link, error) {
	if ownerID == "" {
		return nil, ErrEmptyOwner
	}

	links, err := s.store.ListByOwner(ctx, ownerID, page.Normalize())
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}

	return links, nil
}

func (s *Service) Delete(ctx context.Context, ownerID, code string) error {
	if ownerID == "" {
		return ErrEmptyOwner
	}

	l, err := s.store.ByCode(ctx, code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}

		return fmt.Errorf("delete link: %w", err)
	}

	if l.OwnerID != ownerID {
		return ErrForbidden
	}

	if err := s.store.Delete(ctx, l.ID); err != nil {
		return fmt.Errorf("delete link: %w", err)
	}

	return nil
}
