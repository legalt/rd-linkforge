package config

import "context"

// Loader resolves configuration from its source. The interface lets tests
// supply a fixed Config without mutating the process environment.
type Loader interface {
	Load(ctx context.Context) (Config, error)
}

// NoopLoader satisfies Loader and returns an empty configuration. Implemented
// in M3.
type NoopLoader struct{}

var _ Loader = NoopLoader{}

func (NoopLoader) Load(context.Context) (Config, error) { return Config{}, nil }
