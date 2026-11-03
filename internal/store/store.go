package store

import (
	"context"
	"errors"
	"slices"

	"github.com/skskuzan/rd-linkforge/internal/link"
)

var ErrDuplicateID = errors.New("store: duplicated id")

type Store struct {
	byID   map[uint64]link.Link
	byCode map[string]uint64
}

var _ link.Store = (*Store)(nil)

func New() *Store {
	return &Store{
		byID:   make(map[uint64]link.Link),
		byCode: make(map[string]uint64),
	}
}

func (s *Store) Create(_ context.Context, l link.Link) (link.Link, error) {
	if _, ok := s.byID[l.ID]; ok {
		return link.Link{}, ErrDuplicateID
	}
	if _, ok := s.byCode[l.Code]; ok {
		return link.Link{}, link.ErrAliasTaken
	}

	s.byID[l.ID] = l
	s.byCode[l.Code] = l.ID

	return l, nil
}

func (s *Store) ByCode(_ context.Context, code string) (link.Link, error) {
	id, ok := s.byCode[code]
	if !ok {
		return link.Link{}, link.ErrNotFound
	}

	l, ok := s.byID[id]
	if !ok {
		return link.Link{}, link.ErrNotFound
	}

	return l, nil
}

func (s *Store) ListByOwner(_ context.Context, ownerID string, page link.Page) ([]link.Link, error) {
	page = page.Normalize()
	after, err := page.AfterID()
	if err != nil {
		return nil, err
	}

	var out []link.Link
	for _, id := range sortedIDs(s.byID) {
		if id <= after {
			continue
		}

		l := s.byID[id]
		if l.OwnerID != ownerID {
			continue
		}

		out = append(out, l)
		if len(out) == page.Limit {
			break
		}
	}

	return out, nil
}

func (s *Store) Delete(_ context.Context, id uint64) error {
	l, ok := s.byID[id]
	if !ok {
		return link.ErrNotFound
	}

	delete(s.byID, id)
	delete(s.byCode, l.Code)

	return nil
}

func (s *Store) Get(id uint64) (link.Link, bool) {
	l, ok := s.byID[id]
	return l, ok
}

func (s *Store) All() []link.Link {
	ids := sortedIDs(s.byID)
	out := make([]link.Link, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.byID[id])
	}

	return out
}

func (s *Store) Count() int {
	return len(s.byID)
}

func sortedIDs(m map[uint64]link.Link) []uint64 {
	ids := make([]uint64, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	return ids
}
