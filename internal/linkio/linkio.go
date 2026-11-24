// Package linkio reads and writes links without depending on their storage.
package linkio

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/legalt/rd-linkforge/internal/link"
)

// Target is one line of an import file.
type Target struct {
	Line int // 1-based line number in the file
	URL  string
}

// ReadTargets returns trimmed, nonblank lines that do not start with '#'.
func ReadTargets(r io.Reader) ([]Target, error) {
	var targets []Target
	err := readLines(r, func(line int, text string) error {
		text = strings.TrimSpace(text)
		if text != "" && !strings.HasPrefix(text, "#") {
			targets = append(targets, Target{Line: line, URL: text})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read targets: %w", err)
	}
	return targets, nil
}

// ReadLinks reads JSON Lines, with one JSON object per link.
func ReadLinks(r io.Reader) ([]link.Link, error) {
	var links []link.Link
	err := readLines(r, func(record int, text string) error {
		var l link.Link
		if err := json.Unmarshal([]byte(text), &l); err != nil {
			return fmt.Errorf("record %d: %w", record, err)
		}
		if strings.TrimSpace(text) == "null" {
			return fmt.Errorf("record %d: expected a JSON object", record)
		}
		links = append(links, l)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read links: %w", err)
	}
	return links, nil
}

// readLines uses a buffered reader so lines are not limited by Scanner's token size.
func readLines(r io.Reader, visit func(int, string) error) error {
	reader := bufio.NewReader(r)
	for line := 1; ; line++ {
		text, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read line %d: %w", line, err)
		}
		if len(text) > 0 {
			if visitErr := visit(line, text); visitErr != nil {
				return visitErr
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}

// WriteLinks writes JSON Lines, with one link per line.
func WriteLinks(w io.Writer, links []link.Link) error {
	encoder := json.NewEncoder(w)
	for i, l := range links {
		if err := encoder.Encode(l); err != nil {
			return fmt.Errorf("write record %d: %w", i+1, err)
		}
	}
	return nil
}

// WriteCSV writes a header and one row per link, with RFC 3339 times in UTC.
func WriteCSV(w io.Writer, links []link.Link) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "code", "target", "owner_id", "created_at", "expires_at"}); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, l := range links {
		expires := ""
		if !l.ExpiresAt.IsZero() {
			expires = l.ExpiresAt.UTC().Format(time.RFC3339)
		}
		if err := writer.Write([]string{
			strconv.FormatUint(l.ID, 10), l.Code, l.Target, l.OwnerID,
			l.CreatedAt.UTC().Format(time.RFC3339), expires,
		}); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV: %w", err)
	}
	return nil
}

// LoadFile reads a JSON Lines file. Missing files wrap fs.ErrNotExist.
func LoadFile(path string) (links []link.Link, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %q: %w", path, closeErr))
		}
	}()
	links, err = ReadLinks(file)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", path, err)
	}
	return links, nil
}

// SaveFile creates or replaces a JSON Lines file and checks its Close error.
func SaveFile(path string, links []link.Link) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %q: %w", path, closeErr))
		}
	}()
	if err := WriteLinks(file, links); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	return nil
}
