package link

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	minAliasLen = 3
	maxAliasLen = 32
)

type AliasError struct {
	Alias string
	Err   error
}

func (e *AliasError) Error() string {
	return fmt.Sprintf("link: alias %q: %v", e.Alias, e.Err)
}

func (e *AliasError) Unwrap() error {
	return e.Err
}

func checkAlias(alias string) error {
	if !validAlias(alias) {
		return &AliasError{Alias: alias, Err: ErrInvalidAlias}
	}
	if reservedAlias(alias) {
		return &AliasError{Alias: alias, Err: ErrReservedAlias}
	}

	return nil
}

func validAlias(alias string) bool {
	n := 0
	for _, r := range alias {
		if !aliasChar(r) {
			return false
		}
		n++
	}

	return n >= minAliasLen && n <= maxAliasLen
}

func aliasChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func reservedAlias(alias string) bool {
	switch strings.ToLower(alias) {
	case "api", "ws", "healthz", "readyz", "metrics", "admin":
		return true
	default:
		return false
	}
}
