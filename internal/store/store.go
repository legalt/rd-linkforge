package store

import (
	"errors"
	"slices"

	"github.com/skskuzan/rd-linkforge/internal/link"
)

var ErrDuplicateID = errors.New("store: duplicated id")

type Store struct {
	byID map[uint64]link.Link
}

func New() Store {
	return Store{
		byID: make(map[uint64]link.Link),
	}
}

func Add(s Store, l link.Link) error {
	if _, ok := Get(s, uint64(l.ID)); ok {
		return ErrDuplicateID
	}

	s.byID[uint64(l.ID)] = l

	return nil
}

func Get(s Store, id uint64) (link.Link, bool) {
	v, ok := s.byID[id]

	return v, ok
}

func All(s Store) []link.Link {
	links := make([]link.Link, 0, len(s.byID))

	for _, val := range s.byID {
		links = append(links, val)
	}

	slices.SortFunc(links, func(a, b link.Link) int {
		if int(a.ID) > int(b.ID) {
			return 1
		}

		return -1
	})

	return links
}

func Count(s Store) int {
	return len(s.byID)
}
