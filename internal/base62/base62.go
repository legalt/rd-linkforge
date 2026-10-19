package base62

import (
	"errors"
	"math"
	"strings"
)

var (
	ErrEmpty            = errors.New("base62: empty code")
	ErrInvalidCharacter = errors.New("base62: invalid character")
	ErrOverflow         = errors.New("base62: code is too long")
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const baseValue uint64 = 62

// Encode renders a number with no padding.
func Encode(v uint64) string {
	if v == 0 {
		return "0"
	}

	var buf [11]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = alphabet[v%baseValue]
		v /= baseValue
	}

	return string(buf[i:])
}

// EncodeWidth does the same but left-pads with zeroes to at least width characters. The value itself is never truncated.
func EncodeWidth(v uint64, width int) string {
	res := Encode(v)

	if pad := width - len(res); pad > 0 {
		return strings.Repeat("0", pad) + res
	}

	return res
}

// Decode parses a code back into a number.
func Decode(s string) (uint64, error) {
	var res uint64 = 0

	if s == "" {
		return 0, ErrEmpty
	}
	// A uint64 in base 62 is at most 11 digits.
	if len(s) > 11 {
		return 0, ErrOverflow
	}

	for _, ch := range s {
		index := strings.IndexRune(alphabet, ch)
		if index == -1 {
			return 0, ErrInvalidCharacter
		}

		digit := uint64(index)
		if res > (math.MaxUint64-digit)/baseValue {
			return 0, ErrOverflow
		}
		res = res*baseValue + digit
	}

	return res, nil
}
