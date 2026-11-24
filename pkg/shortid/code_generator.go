package shortid

import (
	"fmt"
	"io"
	"sync"
)

const codeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type CodeGenerator struct {
	mu      sync.Mutex
	random  io.Reader
	counter uint16
}

var _ Generator = (*CodeGenerator)(nil)

// NewCodeGenerator creates a generator backed by random.
func NewCodeGenerator(random io.Reader) *CodeGenerator {
	return &CodeGenerator{random: random}
}

func (g *CodeGenerator) Next() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	var bytes [3]byte
	if _, err := io.ReadFull(g.random, bytes[:]); err != nil {
		return "", fmt.Errorf("shortid: read random bytes: %w", err)
	}

	random := (uint64(bytes[0])<<16 | uint64(bytes[1])<<8 | uint64(bytes[2])) & ((1 << 19) - 1)
	value := random<<16 | uint64(g.counter)
	g.counter++
	return encodeCode(value), nil
}

func encodeCode(value uint64) string {
	var code [6]byte
	for i := len(code) - 1; i >= 0; i-- {
		code[i] = codeAlphabet[value%62]
		value /= 62
	}
	return string(code[:])
}
