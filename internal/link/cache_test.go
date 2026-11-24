package link

import (
	"context"
	"errors"
	"testing"
	"time"
)

type cacheSpy struct {
	links   map[string]Link
	gets    int
	puts    int
	deletes int
}

func newCacheSpy() *cacheSpy {
	return &cacheSpy{links: make(map[string]Link)}
}

func (c *cacheSpy) Get(code string) (Link, bool) {
	c.gets++
	l, ok := c.links[code]
	return l, ok
}

func (c *cacheSpy) Put(code string, l Link) {
	c.puts++
	c.links[code] = l
}

func (c *cacheSpy) Delete(code string) {
	c.deletes++
	delete(c.links, code)
}

type cacheStore struct {
	link        Link
	byCodeCalls int
	byCodeErr   error
	deleteErr   error
}

func (s *cacheStore) Create(_ context.Context, l Link) (Link, error) { return l, nil }

func (s *cacheStore) ByCode(context.Context, string) (Link, error) {
	s.byCodeCalls++
	return s.link, s.byCodeErr
}

func (s *cacheStore) ListByOwner(context.Context, string, Page) ([]Link, error) { return nil, nil }

func (s *cacheStore) Delete(context.Context, uint64) error { return s.deleteErr }

func TestServiceResolveUsesAndFillsCache(t *testing.T) {
	stored := Link{ID: 1, Code: "code", OwnerID: "owner"}
	store := &cacheStore{link: stored}
	cache := newCacheSpy()
	service := NewService(store, WithCache(cache))

	if got, err := service.Resolve(context.Background(), "code"); err != nil || got != stored {
		t.Fatalf("first Resolve = %+v, %v", got, err)
	}
	if got, err := service.Resolve(context.Background(), "code"); err != nil || got != stored {
		t.Fatalf("cached Resolve = %+v, %v", got, err)
	}
	if store.byCodeCalls != 1 || cache.gets != 2 || cache.puts != 1 {
		t.Errorf("calls: store=%d get=%d put=%d", store.byCodeCalls, cache.gets, cache.puts)
	}
}

func TestServiceResolveDoesNotCacheErrors(t *testing.T) {
	boom := errors.New("boom")
	store := &cacheStore{byCodeErr: boom}
	cache := newCacheSpy()
	service := NewService(store, WithCache(cache))

	if _, err := service.Resolve(context.Background(), "code"); !errors.Is(err, boom) {
		t.Fatalf("Resolve error = %v", err)
	}
	if cache.puts != 0 {
		t.Errorf("cache puts = %d, want 0", cache.puts)
	}
}

func TestServiceChecksCachedExpiry(t *testing.T) {
	now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newCacheSpy()
	cache.links["old"] = Link{Code: "old", ExpiresAt: now}
	service := NewService(nil, WithCache(cache), WithClock(func() time.Time { return now }))

	if _, err := service.Resolve(context.Background(), "old"); !errors.Is(err, ErrExpired) {
		t.Fatalf("Resolve error = %v, want ErrExpired", err)
	}
}

func TestServiceDeleteInvalidatesCacheAfterStore(t *testing.T) {
	stored := Link{ID: 1, Code: "code", OwnerID: "owner"}
	store := &cacheStore{link: stored}
	cache := newCacheSpy()
	cache.links[stored.Code] = stored
	service := NewService(store, WithCache(cache))

	if err := service.Delete(context.Background(), "owner", "code"); err != nil {
		t.Fatal(err)
	}
	if cache.deletes != 1 {
		t.Errorf("cache deletes = %d, want 1", cache.deletes)
	}
}

func TestServiceDeleteLeavesCacheOnFailure(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		owner   string
		store   *cacheStore
		wantErr error
	}{
		{"empty owner", "", &cacheStore{}, ErrEmptyOwner},
		{"lookup", "owner", &cacheStore{byCodeErr: boom}, boom},
		{"forbidden", "other", &cacheStore{link: Link{OwnerID: "owner"}}, ErrForbidden},
		{"delete", "owner", &cacheStore{link: Link{OwnerID: "owner"}, deleteErr: boom}, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cache := newCacheSpy()
			service := NewService(tc.store, WithCache(cache))
			if err := service.Delete(context.Background(), tc.owner, "code"); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Delete error = %v, want %v", err, tc.wantErr)
			}
			if cache.deletes != 0 {
				t.Errorf("cache deletes = %d, want 0", cache.deletes)
			}
		})
	}
}
