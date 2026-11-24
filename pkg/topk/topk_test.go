package topk

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewRejectsInvalidN(t *testing.T) {
	for _, n := range []int{-1, 0} {
		top, err := New(n)
		if top != nil || !errors.Is(err, ErrInvalidN) {
			t.Errorf("New(%d) = %v, %v", n, top, err)
		}
	}
}

func TestTopKResult(t *testing.T) {
	top, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	for i, count := range []int64{5, 1, 9, 3, 9, 7, 2} {
		top.Add("k"+string(rune('0'+i)), count)
	}
	want := []Item{{Key: "k2", Count: 9}, {Key: "k4", Count: 9}, {Key: "k5", Count: 7}}
	if got := top.Result(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Result = %v, want %v", got, want)
	}
	if got := top.Result(); !reflect.DeepEqual(got, want) {
		t.Fatalf("second Result = %v, want %v", got, want)
	}
}

func TestTopKTiesAndDuplicateKeys(t *testing.T) {
	top, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []Item{{"z", 5}, {"y", 5}, {"x", 5}, {"x", 5}, {"a", 4}} {
		top.Add(item.Key, item.Count)
	}
	want := []Item{{Key: "x", Count: 5}, {Key: "x", Count: 5}}
	if got := top.Result(); !reflect.DeepEqual(got, want) {
		t.Errorf("Result = %v, want %v", got, want)
	}
}
