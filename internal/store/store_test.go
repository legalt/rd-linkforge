package store

import (
	"context"
	"errors"
	"testing"

	"github.com/skskuzan/rd-linkforge/internal/link"
)

func TestAdd(t *testing.T) {
	s := New()
	l := link.Link{ID: 7, Code: "keep", Target: "https://a.example"}
	if _, err := s.Create(context.Background(), l); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, ok := s.Get(l.ID)
	if !ok || got != l {
		t.Errorf("Get(%d) = %+v, %v; want %+v, true", l.ID, got, ok, l)
	}
	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}
}

func TestAddDuplicate(t *testing.T) {
	s := New()
	original := link.Link{ID: 7, Code: "keep", Target: "https://a.example"}
	if _, err := s.Create(context.Background(), original); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := s.Create(context.Background(), link.Link{ID: original.ID, Code: "replace", Target: "https://b.example"})
	if !errors.Is(err, ErrDuplicateID) {
		t.Errorf("Create() error = %v, want %v", err, ErrDuplicateID)
	}
	got, ok := s.Get(original.ID)
	if !ok || got != original {
		t.Errorf("Get(%d) = %+v, %v; want the original link", original.ID, got, ok)
	}
	if s.Count() != 1 {
		t.Errorf("Count() = %d, want 1", s.Count())
	}
}

func TestGet(t *testing.T) {
	s := New()
	got, ok := s.Get(42)
	if ok || got != (link.Link{}) {
		t.Errorf("Get(42) = %+v, %v; want zero Link, false", got, ok)
	}
}

func TestAll(t *testing.T) {
	s := New()
	if got := s.All(); len(got) != 0 {
		t.Errorf("All(empty) = %+v, want empty", got)
	}

	in := []link.Link{
		{ID: 2, Code: "b"},
		{ID: 10, Code: "j"},
		{ID: 1, Code: "a"},
		{ID: 7, Code: "g"},
	}
	for _, l := range in {
		if _, err := s.Create(context.Background(), l); err != nil {
			t.Fatalf("Create(%d): %v", l.ID, err)
		}
	}

	want := []link.Link{
		{ID: 1, Code: "a"},
		{ID: 2, Code: "b"},
		{ID: 7, Code: "g"},
		{ID: 10, Code: "j"},
	}
	got := s.All()
	if len(got) != len(want) {
		t.Fatalf("All() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if s.Count() != len(want) {
		t.Errorf("Count() = %d, want %d", s.Count(), len(want))
	}
}
