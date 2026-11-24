package shortid

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestCodeGeneratorControlValues(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		calls  int
		wanted []string
	}{
		{"counter", bytes.Repeat([]byte{0, 0, 1}, 3), 3, []string{"000H32", "000H33", "000H34"}},
		{"maximum random", []byte{0xff, 0xff, 0xff}, 1, []string{"bVJgVU"}},
		{"zero random", bytes.Repeat([]byte{0, 0, 0}, 2), 2, []string{"000000", "000001"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			generator := NewCodeGenerator(bytes.NewReader(tc.input))
			for i := range tc.calls {
				got, err := generator.Next()
				if err != nil || got != tc.wanted[i] {
					t.Fatalf("Next %d = %q, %v; want %q", i, got, err, tc.wanted[i])
				}
			}
		})
	}
}

func TestCodeGeneratorReadErrorDoesNotAdvanceCounter(t *testing.T) {
	boom := errors.New("boom")
	reader := &failOnceReader{err: boom, data: []byte{0, 0, 0}}
	generator := NewCodeGenerator(reader)

	if code, err := generator.Next(); code != "" || !errors.Is(err, boom) {
		t.Fatalf("failed Next = %q, %v", code, err)
	}
	if code, err := generator.Next(); code != "000000" || err != nil {
		t.Fatalf("Next after failure = %q, %v", code, err)
	}
}

func TestCodeGeneratorCounterWraps(t *testing.T) {
	generator := NewCodeGenerator(zeroReader{})
	for range 1 << 16 {
		if _, err := generator.Next(); err != nil {
			t.Fatal(err)
		}
	}
	if code, err := generator.Next(); code != "000000" || err != nil {
		t.Fatalf("Next after wrap = %q, %v", code, err)
	}
}

func TestNoopImplementations(t *testing.T) {
	codec := NoopCodec{}
	if encoded := codec.Encode(1); encoded != "" {
		t.Errorf("NoopCodec.Encode = %q", encoded)
	}
	if decoded, err := codec.Decode("1"); decoded != 0 || err != nil {
		t.Errorf("NoopCodec.Decode = %d, %v", decoded, err)
	}
	if code, err := (NoopGenerator{}).Next(); code != "" || err != nil {
		t.Errorf("NoopGenerator.Next = %q, %v", code, err)
	}
}

type failOnceReader struct {
	err  error
	data []byte
}

func (r *failOnceReader) Read(p []byte) (int, error) {
	if r.err != nil {
		err := r.err
		r.err = nil
		return 0, err
	}
	return bytes.NewReader(r.data).Read(p)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

var _ io.Reader = (*failOnceReader)(nil)
var _ io.Reader = zeroReader{}
