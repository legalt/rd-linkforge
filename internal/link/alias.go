package link

import (
	"fmt"
	"strings"
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
	if len(alias) < minAliasLen || len(alias) > maxAliasLen {
		return false
	}
	for i := range len(alias) {
		if !aliasChar(alias[i]) {
			return false
		}
	}

	return true
}

func aliasChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}

func reservedAlias(alias string) bool {
	switch strings.ToLower(alias) {
	case "api", "ws", "healthz", "readyz", "metrics", "admin":
		return true
	default:
		return false
	}
}
