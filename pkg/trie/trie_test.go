package trie

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTrieInsertContains(t *testing.T) {
	tree := New()
	if tree.Insert("") || tree.Contains("") {
		t.Fatal("empty string must not be a word")
	}
	if !tree.Insert("doc") || !tree.Contains("doc") {
		t.Fatal("inserted word is missing")
	}
	if tree.Insert("doc") {
		t.Fatal("duplicate insert succeeded")
	}
	if tree.Contains("do") || tree.Contains("dog") {
		t.Fatal("prefix or unknown word reported present")
	}
}

func TestTrieWithPrefix(t *testing.T) {
	tree := New()
	for _, word := range []string{"docs", "doc", "dog", "do-it", "day", "api", "dock"} {
		tree.Insert(word)
	}

	tests := []struct {
		prefix string
		limit  int
		want   []string
	}{
		{"do", 0, []string{"do-it", "doc", "dock", "docs", "dog"}},
		{"do", -1, []string{"do-it", "doc", "dock", "docs", "dog"}},
		{"do", 2, []string{"do-it", "doc"}},
		{"doc", 0, []string{"doc", "dock", "docs"}},
		{"", 0, []string{"api", "day", "do-it", "doc", "dock", "docs", "dog"}},
		{"unknown", 0, []string{}},
	}
	for _, tc := range tests {
		if got := tree.WithPrefix(tc.prefix, tc.limit); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("WithPrefix(%q, %d) = %v, want %v", tc.prefix, tc.limit, got, tc.want)
		}
	}
}

func TestTrieDelete(t *testing.T) {
	tree := New()
	for _, word := range []string{"do", "doc", "dog", "cat"} {
		tree.Insert(word)
	}
	if tree.Delete("") || tree.Delete("missing") || tree.Delete("d") {
		t.Fatal("deleting an absent word succeeded")
	}
	if !tree.Delete("do") || tree.Contains("do") {
		t.Fatal("word prefix was not deleted")
	}
	if !tree.Contains("doc") || !tree.Contains("dog") {
		t.Fatal("deleting prefix removed longer words")
	}
	if !tree.Delete("doc") || !tree.Delete("dog") || !tree.Delete("cat") {
		t.Fatal("existing word was not deleted")
	}
	if got := tree.WithPrefix("", 0); len(got) != 0 {
		t.Errorf("words after deletion = %v", got)
	}
}

func BenchmarkTrieWithPrefix(b *testing.B) {
	tree := New()
	for i := range 10_000 {
		tree.Insert(fmt.Sprintf("word-%05d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		tree.WithPrefix("word-099", 0)
	}
}
