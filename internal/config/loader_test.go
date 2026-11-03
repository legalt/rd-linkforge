package config

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func validEnv(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"LF_PG_DSN":     "postgres://user:pg-password@db/app",
		"LF_MONGO_URI":  "mongodb://user:mongo-password@db/app",
		"LF_JWT_SECRET": strings.Repeat("secret", 6),
	}
}

func envLookup(t *testing.T, values map[string]string) lookUpFn {
	t.Helper()
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestEnvLoaderDefaults(t *testing.T) {
	for _, blank := range []string{"", " \t\n"} {
		t.Run("blank="+blank, func(t *testing.T) {
			env := validEnv(t)
			for _, key := range []string{"LF_HTTP_ADDR", "LF_ADMIN_ADDR", "LF_TCP_ADDR", "LF_CACHE_SIZE", "LF_LOG_LEVEL"} {
				env[key] = blank
			}
			got, err := NewEnvLoader(envLookup(t, env)).Load(context.Background())
			want := Config{
				HTTPAddr: ":8080", AdminAddr: "127.0.0.1:6060", TCPAddr: ":7070",
				PGDSN: env["LF_PG_DSN"], MongoURI: env["LF_MONGO_URI"], JWTSecret: env["LF_JWT_SECRET"],
				CacheSize: 100000, LogLevel: slog.LevelInfo,
			}
			if err != nil || got != want {
				t.Fatalf("Load = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestEnvLoaderOverrides(t *testing.T) {
	env := validEnv(t)
	env["LF_HTTP_ADDR"] = " localhost:1 "
	env["LF_ADMIN_ADDR"] = " [::1]:65535\t"
	env["LF_TCP_ADDR"] = " :9090\n"
	env["LF_CACHE_SIZE"] = " 42 "
	env["LF_LOG_LEVEL"] = " dEbUg "
	for _, key := range []string{"LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET"} {
		env[key] = " \t" + env[key] + "\n "
	}
	got, err := NewEnvLoader(envLookup(t, env)).Load(context.Background())
	want := Config{
		HTTPAddr: "localhost:1", AdminAddr: "[::1]:65535", TCPAddr: ":9090",
		PGDSN: strings.TrimSpace(env["LF_PG_DSN"]), MongoURI: strings.TrimSpace(env["LF_MONGO_URI"]),
		JWTSecret: strings.TrimSpace(env["LF_JWT_SECRET"]), CacheSize: 42, LogLevel: slog.LevelDebug,
	}
	if err != nil || got != want {
		t.Fatalf("Load = %+v, %v; want %+v", got, err, want)
	}
}

func TestEnvLoaderValidation(t *testing.T) {
	tests := []struct {
		name, key, value string
		unset            bool
		want             error
	}{
		{"missing DSN", "LF_PG_DSN", "", true, ErrMissing},
		{"empty DSN", "LF_PG_DSN", "", false, ErrMissing},
		{"blank DSN", "LF_PG_DSN", " \t", false, ErrMissing},
		{"missing URI", "LF_MONGO_URI", "", true, ErrMissing},
		{"blank URI", "LF_MONGO_URI", " \n", false, ErrMissing},
		{"missing secret", "LF_JWT_SECRET", "", true, ErrMissing},
		{"blank secret", "LF_JWT_SECRET", " \t", false, ErrMissing},
		{"short secret", "LF_JWT_SECRET", strings.Repeat("x", 31), false, ErrInvalid},
		{"missing port", "LF_HTTP_ADDR", "localhost", false, ErrInvalid},
		{"named port", "LF_HTTP_ADDR", "localhost:http", false, ErrInvalid},
		{"empty port", "LF_HTTP_ADDR", "localhost:", false, ErrInvalid},
		{"zero port", "LF_ADMIN_ADDR", ":0", false, ErrInvalid},
		{"negative port", "LF_ADMIN_ADDR", ":-1", false, ErrInvalid},
		{"port too large", "LF_TCP_ADDR", ":65536", false, ErrInvalid},
		{"unbracketed IPv6", "LF_TCP_ADDR", "::1:8080", false, ErrInvalid},
		{"invalid cache", "LF_CACHE_SIZE", "lots", false, ErrInvalid},
		{"fractional cache", "LF_CACHE_SIZE", "1.5", false, ErrInvalid},
		{"cache overflow", "LF_CACHE_SIZE", strings.Repeat("9", 30), false, ErrInvalid},
		{"zero cache", "LF_CACHE_SIZE", "0", false, ErrInvalid},
		{"negative cache", "LF_CACHE_SIZE", "-1", false, ErrInvalid},
		{"invalid level", "LF_LOG_LEVEL", "trace", false, ErrInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := validEnv(t)
			env[tc.key] = tc.value
			if tc.unset {
				delete(env, tc.key)
			}
			got, err := NewEnvLoader(envLookup(t, env)).Load(context.Background())
			if !errors.Is(err, tc.want) || !strings.HasPrefix(err.Error(), tc.key+":") {
				t.Fatalf("Load error = %v; want %s: %v", err, tc.key, tc.want)
			}
			if got != (Config{}) {
				t.Errorf("invalid configuration returned a nonzero Config")
			}
		})
	}
}

func TestEnvLoaderJoinsErrorsAndHidesSecrets(t *testing.T) {
	env := validEnv(t)
	env["LF_HTTP_ADDR"] = "bad address"
	env["LF_CACHE_SIZE"] = "0"
	env["LF_JWT_SECRET"] = "short-secret"
	_, err := NewEnvLoader(envLookup(t, env)).Load(context.Background())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Load error = %v", err)
	}
	for _, key := range []string{"LF_HTTP_ADDR", "LF_CACHE_SIZE", "LF_JWT_SECRET"} {
		if !strings.Contains(err.Error(), key+":") {
			t.Errorf("missing %s in %v", key, err)
		}
	}
	for _, key := range []string{"LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET"} {
		if strings.Contains(err.Error(), env[key]) {
			t.Errorf("error exposes %s", key)
		}
	}
	delete(env, "LF_PG_DSN")
	delete(env, "LF_MONGO_URI")
	_, err = NewEnvLoader(envLookup(t, env)).Load(context.Background())
	if !errors.Is(err, ErrMissing) || !errors.Is(err, ErrInvalid) {
		t.Fatalf("joined error must wrap both sentinels: %v", err)
	}
	for _, key := range []string{"LF_HTTP_ADDR", "LF_PG_DSN", "LF_MONGO_URI", "LF_JWT_SECRET", "LF_CACHE_SIZE"} {
		if !strings.Contains(err.Error(), key+":") {
			t.Errorf("missing %s in %v", key, err)
		}
	}
}

func TestEnvLoaderCredentialsOnlyValidatePresenceAndBytes(t *testing.T) {
	env := validEnv(t)
	env["LF_PG_DSN"] = "opaque DSN"
	env["LF_MONGO_URI"] = "opaque URI"
	env["LF_JWT_SECRET"] = strings.Repeat("я", 16) // 32 bytes, 16 runes.
	if _, err := NewEnvLoader(envLookup(t, env)).Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestEnvLoaderContext(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	for _, ctx := range []context.Context{canceled, expired} {
		t.Run(ctx.Err().Error(), func(t *testing.T) {
			loader := NewEnvLoader(func(string) (string, bool) {
				t.Fatal("canceled Load consulted the environment")
				return "", false
			})
			got, err := loader.Load(ctx)
			if err != ctx.Err() || got != (Config{}) {
				t.Fatalf("Load = %+v, %v; want zero Config, %v", got, err, ctx.Err())
			}
		})
	}
	if NewEnvLoader(nil).lookup == nil {
		t.Fatal("nil lookup must default to os.LookupEnv")
	}
}

func TestParseLevel(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  slog.Level
	}{
		{" DEBUG ", slog.LevelDebug}, {"iNfO", slog.LevelInfo},
		{"\tWarn\n", slog.LevelWarn}, {"ERROR", slog.LevelError},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseLevel(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("ParseLevel = %v, %v", got, err)
			}
		})
	}
	for _, input := range []string{"", " ", "trace", "warning", "INFO+1", "4"} {
		t.Run("invalid="+input, func(t *testing.T) {
			if _, err := ParseLevel(input); !errors.Is(err, ErrInvalid) {
				t.Fatalf("ParseLevel error = %v", err)
			}
		})
	}
}

func TestConfigLogValue(t *testing.T) {
	env := validEnv(t)
	cfg, err := NewEnvLoader(envLookup(t, env)).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	attrs := cfg.LogValue().Group()
	if len(attrs) != 8 {
		t.Fatalf("LogValue has %d attributes; want 8", len(attrs))
	}
	wantKeys := []string{"http_addr", "admin_addr", "tcp_addr", "pg_dsn", "mongo_uri", "jwt_secret", "cache_size", "log_level"}
	for i, attr := range attrs {
		if attr.Key != wantKeys[i] {
			t.Errorf("attribute %d = %s; want %s", i, attr.Key, wantKeys[i])
		}
	}
	var output bytes.Buffer
	slog.New(slog.NewTextHandler(&output, nil)).Info("loaded", "config", cfg)
	if strings.Count(output.String(), "[redacted]") != 3 {
		t.Errorf("missing redactions: %s", &output)
	}
	for _, secret := range env {
		if strings.Contains(output.String(), secret) {
			t.Error("configuration log exposes a credential")
		}
	}
}
