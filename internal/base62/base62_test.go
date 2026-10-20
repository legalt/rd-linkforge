package base62

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := map[uint64]string{
		0: "0", 9: "9", 10: "A", 35: "Z", 36: "a", 61: "z",
		62: "10", 3843: "zz", 3844: "100", 123456: "W7E",
	}
	for in, want := range tests {
		if got := Encode(in); got != want {
			t.Errorf("Encode(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestEncodeWidth(t *testing.T) {
	if got := EncodeWidth(61, 6); got != "00000z" {
		t.Errorf("EncodeWidth(61, 6) = %q, want \"00000z\"", got)
	}
	if got := EncodeWidth(3844, 2); got != "100" {
		t.Errorf("EncodeWidth(3844, 2) = %q - the value must not be truncated", got)
	}
}

func TestDecode(t *testing.T) {
	for _, v := range []uint64{0, 1, 61, 62, 3843, 123456, 1 << 40, ^uint64(0)} {
		code := Encode(v)
		got, err := Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q): %v", code, err)
		}
		if got != v {
			t.Errorf("Decode(Encode(%d)) = %d", v, got)
		}
	}
	if got, err := Decode("00000z"); err != nil || got != 61 {
		t.Errorf("Decode(\"00000z\") = %d, %v; want 61, nil", got, err)
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := map[string]error{
		"":                      ErrEmpty,
		"ab-cd":                 ErrInvalidCharacter,
		"ab_cd":                 ErrInvalidCharacter,
		"hello world":           ErrInvalidCharacter,
		strings.Repeat("z", 11): ErrOverflow,
		strings.Repeat("z", 12): ErrOverflow,
		strings.Repeat("0", 12): ErrOverflow,
		"!!!!!!!!!!!!":          ErrOverflow,
	}
	for in, want := range tests {
		if _, err := Decode(in); !errors.Is(err, want) {
			t.Errorf("Decode(%q) returned %v, want %v", in, err, want)
		}
	}

	// A multi-byte character must be rejected, not split into bytes.
	if _, err := Decode("café"); !errors.Is(err, ErrInvalidCharacter) {
		t.Errorf("Decode of a non-ASCII code returned %v, want ErrInvalidCharacter", err)
	}
}

func TestEncodeRecursive(t *testing.T) {
	for _, v := range []uint64{0, 1, 9, 10, 61, 62, 3843, 3844, 123456, 1 << 40, math.MaxUint64} {
		if got, want := EncodeRecursive(v), Encode(v); got != want {
			t.Errorf("EncodeRecursive(%d) = %q, want %q", v, got, want)
		}
	}
}

func TestDecodeRange(t *testing.T) {
	tests := []struct {
		in      string
		want    uint64
		wantErr error
	}{
		{"0000000000z", 61, nil},
		{"LygHa16AHYF", math.MaxUint64, nil},
		{"LygHa16AHYG", 0, ErrOverflow},
		{"zzzzzzzzzzz", 0, ErrOverflow},
		{"00000000000z", 0, ErrOverflow},
	}
	for _, tc := range tests {
		got, err := Decode(tc.in)
		if got != tc.want || !errors.Is(err, tc.wantErr) {
			t.Errorf("Decode(%q) = %d, %v; want %d, %v", tc.in, got, err, tc.want, tc.wantErr)
		}
	}
}
