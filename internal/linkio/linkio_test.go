package linkio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/legalt/rd-linkforge/internal/link"
)

func sampleLinks(t *testing.T) []link.Link {
	t.Helper()
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return []link.Link{
		{ID: 1, Code: "000001", Target: "https://go.dev", OwnerID: "cli", CreatedAt: created},
		{ID: 2, Code: "docs", Target: "https://example.com/a,b", OwnerID: "other", CreatedAt: created, ExpiresAt: created.Add(time.Hour)},
	}
}

func TestReadTargets(t *testing.T) {
	got, err := ReadTargets(strings.NewReader(" # comment\n\n https://go.dev \r\n\t\n# ignored\n https://example.com/#fragment"))
	want := []Target{{Line: 3, URL: "https://go.dev"}, {Line: 6, URL: "https://example.com/#fragment"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadTargets = %+v, %v; want %+v", got, err, want)
	}
	if got, err := ReadTargets(strings.NewReader("")); err != nil || len(got) != 0 {
		t.Fatalf("empty ReadTargets = %+v, %v", got, err)
	}
	longURL := "https://example.com/" + strings.Repeat("a", 100000)
	if got, err := ReadTargets(strings.NewReader(longURL)); err != nil || len(got) != 1 || got[0].URL != longURL {
		t.Fatal("long target was not read intact", err)
	}
}

func TestLinksRoundTrip(t *testing.T) {
	want := sampleLinks(t)
	var output bytes.Buffer
	if err := WriteLinks(&output, want); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != len(want) {
		t.Fatalf("wrote %d lines; want %d", len(lines), len(want))
	}
	for i, l := range want {
		encoded, err := json.Marshal(l)
		if err != nil || lines[i] != string(encoded) {
			t.Fatalf("line %d does not match encoding/json: %v", i+1, err)
		}
	}
	got, err := ReadLinks(&output)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadLinks = %+v, %v; want %+v", got, err, want)
	}
	if got, err := ReadLinks(strings.NewReader("")); err != nil || len(got) != 0 {
		t.Fatalf("empty ReadLinks = %+v, %v", got, err)
	}
}

func TestReadLinksBadRecord(t *testing.T) {
	for _, record := range []string{"{", `{"ID":"bad"}`, "null", "[]", "{} {}", "", "true"} {
		t.Run(record, func(t *testing.T) {
			if _, err := ReadLinks(strings.NewReader("{}\n{}\n" + record + "\n")); err == nil || !strings.Contains(err.Error(), "record 3:") {
				t.Fatalf("error = %v; want record 3", err)
			}
		})
	}
	_, err := ReadLinks(strings.NewReader("{\n"))
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("JSON error was not wrapped: %v", err)
	}
}

func TestWriteCSV(t *testing.T) {
	links := sampleLinks(t)
	zone := time.FixedZone("test", 2*60*60)
	links[0].CreatedAt = links[0].CreatedAt.In(zone)
	links[1].ExpiresAt = links[1].ExpiresAt.In(zone)
	var output bytes.Buffer
	if err := WriteCSV(&output, links); err != nil {
		t.Fatal(err)
	}
	want := "id,code,target,owner_id,created_at,expires_at\n" +
		"1,000001,https://go.dev,cli,2026-01-02T03:04:05Z,\n" +
		"2,docs,\"https://example.com/a,b\",other,2026-01-02T03:04:05Z,2026-01-02T04:04:05Z\n"
	if output.String() != want {
		t.Fatalf("CSV = %q; want %q", output.String(), want)
	}
	output.Reset()
	if err := WriteCSV(&output, nil); err != nil || output.String() != strings.SplitN(want, "\n", 2)[0]+"\n" {
		t.Fatalf("empty CSV = %q, %v", output.String(), err)
	}
}

type failingIO struct{ err error }

func (f failingIO) Read([]byte) (int, error)  { return 0, f.err }
func (f failingIO) Write([]byte) (int, error) { return 0, f.err }

func TestIOErrorsAreWrapped(t *testing.T) {
	boom := errors.New("I/O failure")
	links := sampleLinks(t)
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"targets", func() error { _, err := ReadTargets(failingIO{boom}); return err }},
		{"read links", func() error { _, err := ReadLinks(failingIO{boom}); return err }},
		{"write links", func() error { return WriteLinks(failingIO{boom}, links) }},
		{"CSV flush", func() error { return WriteCSV(failingIO{boom}, links) }},
		{"CSV row", func() error {
			return WriteCSV(failingIO{boom}, []link.Link{{Target: strings.Repeat("a", 10000)}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, boom) {
				t.Fatalf("error = %v; want wrapped I/O failure", err)
			}
		})
	}
	invalidTime := []link.Link{{CreatedAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}}
	if err := WriteLinks(io.Discard, invalidTime); err == nil {
		t.Fatal("expected JSON encoding error")
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.jsonl")
	want := sampleLinks(t)
	if err := SaveFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadFile = %+v, %v; want %+v", got, err, want)
	}
	if err := SaveFile(path, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadFile(path); err != nil || len(got) != 0 {
		t.Fatalf("SaveFile did not replace contents: %+v, %v", got, err)
	}
	missing := filepath.Join(dir, "missing.jsonl")
	if _, err := LoadFile(missing); !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), missing) {
		t.Fatalf("missing file error = %v", err)
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("invalid file error = %v", err)
	}
	if err := SaveFile(dir, want); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("save to directory error = %v", err)
	}
	invalidTime := []link.Link{{CreatedAt: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}}
	if err := SaveFile(path, invalidTime); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("save invalid link error = %v", err)
	}
}
