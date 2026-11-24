package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/legalt/rd-linkforge/internal/link"
	"github.com/legalt/rd-linkforge/internal/linkio"
)

func lookupEnv(t *testing.T, values map[string]string) func(string) (string, bool) {
	t.Helper()
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runTest(t *testing.T, args []string, env map[string]string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, lookupEnv(t, env), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestImportThenExport(t *testing.T) {
	dir := t.TempDir()
	input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
	writeTestFile(t, input, "# links\n https://go.dev \n\nhttps://example.com\n")
	code, stdout, stderr := runTest(t, []string{"import", "-in", input, "-db", database}, nil)
	if code != 0 || stdout != "imported: 2\nskipped: 0\n" {
		t.Fatalf("import = %d, %q, %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "import started") || !strings.Contains(stderr, "import finished") || strings.Contains(stderr, "target imported") {
		t.Fatalf("unexpected info logs: %s", stderr)
	}
	code, stdout, stderr = runTest(t, []string{"export", "-db", database}, nil)
	if code != 0 || stderr != "" {
		t.Fatalf("export = %d, %q", code, stderr)
	}
	rows, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
	if err != nil || len(rows) != 3 {
		t.Fatalf("CSV = %q, %v", stdout, err)
	}
	if strings.Join(rows[0], ",") != "id,code,target,owner_id,created_at,expires_at" ||
		rows[1][0] != "1" || rows[1][1] != "000001" || rows[1][2] != "https://go.dev" ||
		rows[1][3] != "cli" || rows[1][4] == "" || rows[1][5] != "" || rows[2][0] != "2" {
		t.Fatalf("unexpected CSV: %+v", rows)
	}
	output := filepath.Join(dir, "links.csv")
	wantCSV := stdout
	code, stdout, stderr = runTest(t, []string{"export", "-db", database, "-out", output}, nil)
	if code != 0 || stdout != "" || stderr != "" || readTestFile(t, output) != wantCSV {
		t.Fatalf("export to file = %d, %q, %q", code, stdout, stderr)
	}

	writeTestFile(t, input, "https://example.org\n")
	code, stdout, stderr = runTest(t, []string{"import", "-in", input, "-db", database}, nil)
	if code != 0 || stdout != "imported: 1\nskipped: 0\n" {
		t.Fatalf("second import = %d, %q, %q", code, stdout, stderr)
	}
	links, err := linkio.LoadFile(database)
	if err != nil || len(links) != 3 || links[2].ID != 3 || links[2].Code != "000003" {
		t.Fatalf("second import database = %+v, %v", links, err)
	}
}

func TestImportUsesLargestIDAndPreservesOtherOwners(t *testing.T) {
	dir := t.TempDir()
	input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
	existing := []link.Link{{ID: 42, Code: "answer", OwnerID: "other"}, {ID: 3, Code: "three", OwnerID: "cli"}}
	if err := linkio.SaveFile(database, existing); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, input, "https://go.dev")
	code, _, stderr := runTest(t, []string{"import", "-in", input, "-db", database}, nil)
	if code != 0 {
		t.Fatalf("import = %d, %s", code, stderr)
	}
	links, err := linkio.LoadFile(database)
	if err != nil || len(links) != 3 || links[1] != existing[0] || links[2].ID != 43 {
		t.Fatalf("database = %+v, %v", links, err)
	}
}

func TestImportSkipsInvalidTargets(t *testing.T) {
	dir := t.TempDir()
	input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
	writeTestFile(t, input, "# comment\nhttps://go.dev\npostgres://user:private-password@host/db\nhttps://user:private-password@example.com\n")
	code, stdout, stderr := runTest(t, []string{"import", "-in", input, "-db", database}, map[string]string{"LF_LOG_LEVEL": " DEBUG "})
	if code != 1 || stdout != "imported: 2\nskipped: 1\n" {
		t.Fatalf("import = %d, %q, %q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "level=WARN") || !strings.Contains(stderr, "line=3") || !strings.Contains(stderr, "error=") ||
		!strings.Contains(stderr, "level=DEBUG") || !strings.Contains(stderr, "target imported") {
		t.Errorf("missing structured logs: %s", stderr)
	}
	if strings.Contains(stderr, "private-password") || strings.Contains(stderr, "postgres://") || strings.Contains(stderr, "https://") {
		t.Error("target credentials appeared in logs")
	}
	links, err := linkio.LoadFile(database)
	if err != nil || len(links) != 2 || links[1].ID != 2 {
		t.Fatalf("valid targets were not saved: %+v, %v", links, err)
	}
}

type cancelOnLog struct {
	bytes.Buffer
	trigger string
	cancel  context.CancelFunc
}

func (w *cancelOnLog) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if strings.Contains(string(p), w.trigger) {
		w.cancel()
	}
	return n, err
}

func TestImportCancellationPreservesDatabase(t *testing.T) {
	for _, tc := range []struct {
		name, trigger, target string
	}{
		{"before import", "", "https://go.dev"},
		{"during read", "import started", "https://go.dev"},
		{"after final success", "target imported", "https://go.dev"},
		{"after final skip", "target skipped", "invalid"},
	} {
		for _, exists := range []bool{false, true} {
			name := tc.name + "/missing"
			if exists {
				name = tc.name + "/existing"
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
				writeTestFile(t, input, tc.target)
				original := "{\"ID\":7,\"Code\":\"keep\",\"OwnerID\":\"other\"}\n"
				if exists {
					writeTestFile(t, database, original)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tc.trigger == "" {
					cancel()
				}
				var stdout bytes.Buffer
				stderr := &cancelOnLog{trigger: tc.trigger, cancel: cancel}
				code := run(ctx, []string{"import", "-in", input, "-db", database},
					lookupEnv(t, map[string]string{"LF_LOG_LEVEL": "debug"}), &stdout, stderr)
				if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), context.Canceled.Error()) {
					t.Fatalf("canceled import = %d, %q, %q", code, &stdout, stderr.String())
				}
				if exists {
					if got := readTestFile(t, database); got != original {
						t.Errorf("canceled import changed database: %q", got)
					}
				} else if _, err := os.Stat(database); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("canceled import created database: %v", err)
				}
			})
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		nil, {"unknown"}, {"import"}, {"import", "-in", "input"}, {"import", "-db", "db"},
		{"import", "-in"}, {"import", "-db"}, {"import", "-unknown"},
		{"import", "-in", "input", "-db", "db", "extra"},
		{"export"}, {"export", "-db"}, {"export", "-db", "db", "-out"}, {"export", "-unknown"},
		{"export", "-db", "db", "extra"}, {"config", "-unknown"}, {"config", "extra"},
		{"version", "-unknown"}, {"version", "extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, stdout, stderr := runTest(t, args, nil)
			if code != 2 || stdout != "" || stderr == "" {
				t.Fatalf("run = %d, %q, %q", code, stdout, stderr)
			}
		})
	}
	code, stdout, stderr := runTest(t, []string{"version"}, map[string]string{"LF_LOG_LEVEL": "secret-invalid-level"})
	if code != 2 || stdout != "" || !strings.Contains(stderr, "LF_LOG_LEVEL") || strings.Contains(stderr, "secret-invalid-level") {
		t.Fatalf("invalid level = %d, %q, %q", code, stdout, stderr)
	}
}

func TestVersionAndHelp(t *testing.T) {
	for _, level := range []string{"", " \t", "warn", "ERROR"} {
		code, stdout, stderr := runTest(t, []string{"version"}, map[string]string{"LF_LOG_LEVEL": level})
		if code != 0 || stdout != "linkctl "+version+"\n" || stderr != "" {
			t.Fatalf("version = %d, %q, %q", code, stdout, stderr)
		}
	}
	for _, command := range []string{"import", "export", "config", "version"} {
		code, stdout, stderr := runTest(t, []string{command, "-h"}, nil)
		if code != 0 || stdout != "" || !strings.Contains(stderr, "Usage") {
			t.Fatalf("help = %d, %q, %q", code, stdout, stderr)
		}
	}
}

func TestConfigCommand(t *testing.T) {
	env := map[string]string{
		"LF_PG_DSN": "postgres://private-password@db", "LF_MONGO_URI": "mongodb://private-password@db",
		"LF_JWT_SECRET": strings.Repeat("private-secret", 3),
	}
	code, stdout, stderr := runTest(t, []string{"config"}, env)
	if code != 0 || stdout != "config ok\n" || strings.Count(stderr, "[redacted]") != 3 {
		t.Fatalf("config = %d, %q, %q", code, stdout, stderr)
	}
	for _, secret := range env {
		if strings.Contains(stderr, secret) {
			t.Error("config log exposes a credential")
		}
	}
	env["LF_JWT_SECRET"] = "short-private-secret"
	env["LF_CACHE_SIZE"] = "0"
	code, stdout, stderr = runTest(t, []string{"config"}, env)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "invalid configuration:") ||
		!strings.Contains(stderr, "LF_JWT_SECRET:") || !strings.Contains(stderr, "LF_CACHE_SIZE:") {
		t.Fatalf("invalid config = %d, %q, %q", code, stdout, stderr)
	}
	for _, key := range []string{"LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET"} {
		if strings.Contains(stderr, env[key]) {
			t.Errorf("config error exposes %s", key)
		}
	}
	code, _, stderr = runTest(t, []string{"config"}, nil)
	if code != 1 || !strings.Contains(stderr, "LF_PG_DSN:") || !strings.Contains(stderr, "LF_MONGO_URI:") || !strings.Contains(stderr, "LF_JWT_SECRET:") {
		t.Fatalf("missing config = %d, %q", code, stderr)
	}
}

func TestFileFailures(t *testing.T) {
	dir := t.TempDir()
	input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
	missing, invalid := filepath.Join(dir, "missing"), filepath.Join(dir, "invalid.jsonl")
	duplicate, exhausted := filepath.Join(dir, "duplicates.jsonl"), filepath.Join(dir, "exhausted.jsonl")
	writeTestFile(t, input, "https://go.dev")
	writeTestFile(t, invalid, "broken")
	for path, links := range map[string][]link.Link{
		database:  nil,
		duplicate: {{ID: 1, Code: "one"}, {ID: 1, Code: "two"}},
		exhausted: {{ID: math.MaxUint64, Code: "last"}},
	} {
		if err := linkio.SaveFile(path, links); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"missing input", []string{"import", "-in", missing, "-db", database}},
		{"directory input", []string{"import", "-in", dir, "-db", database}},
		{"invalid import database", []string{"import", "-in", input, "-db", invalid}},
		{"directory database", []string{"import", "-in", input, "-db", dir}},
		{"duplicate records", []string{"import", "-in", input, "-db", duplicate}},
		{"exhausted IDs", []string{"import", "-in", input, "-db", exhausted}},
		{"save failure", []string{"import", "-in", input, "-db", filepath.Join(missing, "db")}},
		{"missing export database", []string{"export", "-db", missing}},
		{"invalid export database", []string{"export", "-db", invalid}},
		{"create output failure", []string{"export", "-db", database, "-out", dir}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runTest(t, tc.args, nil)
			if code != 1 || stdout != "" || stderr == "" {
				t.Fatalf("run = %d, %q, %q", code, stdout, stderr)
			}
			if tc.name == "missing export database" && !strings.Contains(stderr, missing) {
				t.Errorf("missing file error does not identify the file: %s", stderr)
			}
		})
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestOutputFailure(t *testing.T) {
	dir := t.TempDir()
	input, database := filepath.Join(dir, "targets.txt"), filepath.Join(dir, "links.jsonl")
	writeTestFile(t, input, "https://go.dev")
	writeTestFile(t, database, "")
	for _, args := range [][]string{{"version"}, {"import", "-in", input, "-db", database}, {"export", "-db", database}} {
		t.Run(args[0], func(t *testing.T) {
			var stderr bytes.Buffer
			if code := run(context.Background(), args, lookupEnv(t, nil), failedWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "output unavailable") {
				t.Fatalf("run = %d, %q", code, &stderr)
			}
		})
	}
}
