package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

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

var (
	ErrMissing = errors.New("config: required variable is not set")
	ErrInvalid = errors.New("config: invalid value")
)

type lookUpFn = func(key string) (value string, ok bool)

type EnvLoader struct {
	lookup lookUpFn
}

var _ Loader = (*EnvLoader)(nil)

// NewEnvLoader takes the function that reads variables, so that tests do not have
// to touch the real environment. A nil lookup means os.LookupEnv.
func NewEnvLoader(lookup lookUpFn) *EnvLoader {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	return &EnvLoader{lookup: lookup}
}

func getDefaultConf() Config {
	return Config{
		HTTPAddr:  ":8080",
		AdminAddr: "127.0.0.1:6060",
		TCPAddr:   ":7070",
		CacheSize: 100000,
		LogLevel:  slog.LevelInfo,
	}
}

func (l *EnvLoader) Load(ctx context.Context) (Config, error) {
	if err := ctx.Err(); err != nil {
		return Config{}, err
	}

	config := getDefaultConf()
	var errs []error

	for _, field := range []struct {
		key string
		dst *string
	}{
		{"LF_HTTP_ADDR", &config.HTTPAddr},
		{"LF_ADMIN_ADDR", &config.AdminAddr},
		{"LF_TCP_ADDR", &config.TCPAddr},
	} {
		if val := l.value(field.key); val != "" {
			_, port, err := net.SplitHostPort(val)
			n, portErr := strconv.Atoi(port)
			if err != nil || portErr != nil || n < 1 || n > 65535 {
				errs = append(errs, fmt.Errorf("%s: %w (expected host:port with port 1-65535)", field.key, ErrInvalid))
			} else {
				*field.dst = val
			}
		}
	}

	for _, field := range []struct {
		key string
		dst *string
	}{
		{"LF_PG_DSN", &config.PGDSN},
		{"LF_MONGO_URI", &config.MongoURI},
		{"LF_JWT_SECRET", &config.JWTSecret},
	} {
		*field.dst = l.value(field.key)
		if *field.dst == "" {
			errs = append(errs, fmt.Errorf("%s: %w", field.key, ErrMissing))
		} else if field.key == "LF_JWT_SECRET" && len(*field.dst) < 32 {
			errs = append(errs, fmt.Errorf("%s: %w (expected at least 32 bytes)", field.key, ErrInvalid))
		}
	}

	if val := l.value("LF_CACHE_SIZE"); val != "" {
		n, err := strconv.Atoi(val)
		if err != nil || n <= 0 {
			errs = append(errs, fmt.Errorf("LF_CACHE_SIZE: %w (expected a positive integer)", ErrInvalid))
		} else {
			config.CacheSize = n
		}
	}
	if val := l.value("LF_LOG_LEVEL"); val != "" {
		level, err := ParseLevel(val)
		if err != nil {
			errs = append(errs, fmt.Errorf("LF_LOG_LEVEL: %w", err))
		} else {
			config.LogLevel = level
		}
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (l *EnvLoader) value(key string) string {
	value, ok := l.lookup(key)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

// ParseLevel accepts debug, info, warn and error in any letter case, with spaces around.
// Anything else returns an error that wraps ErrInvalid.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w (expected debug, info, warn or error)", ErrInvalid)
	}
}
