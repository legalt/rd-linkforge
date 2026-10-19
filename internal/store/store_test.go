package store

import (
	"errors"
	"testing"

	"github.com/skskuzan/rd-linkforge/internal/link"
)

func TestAdd(t *testing.T) {
	s := New()
	l := link.Link{ID: 7, Code: "keep", Target: "https://a.example"}
	if err := Add(s, l); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	got, ok := Get(s, l.ID)
	if !ok || got != l {
		t.Errorf("Get(%d) = %+v, %v; want %+v, true", l.ID, got, ok, l)
	}
	if Count(s) != 1 {
		t.Errorf("Count() = %d, want 1", Count(s))
	}
}

func TestAddDuplicate(t *testing.T) {
	s := New()
	original := link.Link{ID: 7, Code: "keep", Target: "https://a.example"}
	if err := Add(s, original); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	err := Add(s, link.Link{ID: original.ID, Code: "replace", Target: "https://b.example"})
	if !errors.Is(err, ErrDuplicateID) {
		t.Errorf("Add() error = %v, want %v", err, ErrDuplicateID)
	}
	got, ok := Get(s, original.ID)
	if !ok || got != original {
		t.Errorf("Get(%d) = %+v, %v; want the original link", original.ID, got, ok)
	}
	if Count(s) != 1 {
		t.Errorf("Count() = %d, want 1", Count(s))
	}
}

func TestGet(t *testing.T) {
	got, ok := Get(New(), 42)
	if ok || got != (link.Link{}) {
		t.Errorf("Get(42) = %+v, %v; want zero Link, false", got, ok)
	}
}

func TestAll(t *testing.T) {
	s := New()
	if got := All(s); len(got) != 0 {
		t.Errorf("All(empty) = %+v, want empty", got)
	}

	in := []link.Link{
		{ID: 2, Code: "b"},
		{ID: 10, Code: "j"},
		{ID: 1, Code: "a"},
		{ID: 7, Code: "g"},
	}
	for _, l := range in {
		if err := Add(s, l); err != nil {
			t.Fatalf("Add(%d): %v", l.ID, err)
		}
	}

	want := []link.Link{
		{ID: 1, Code: "a"},
		{ID: 2, Code: "b"},
		{ID: 7, Code: "g"},
		{ID: 10, Code: "j"},
	}
	got := All(s)
	if len(got) != len(want) {
		t.Fatalf("All() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if Count(s) != len(want) {
		t.Errorf("Count() = %d, want %d", Count(s), len(want))
	}
}
