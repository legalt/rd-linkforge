package link

import (
	"context"
	"testing"
	"time"
)

func TestServiceContext(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	for _, ctx := range []context.Context{canceled, expired} {
		t.Run(ctx.Err().Error(), func(t *testing.T) {
			// A nil store panics if any method gets past the context check.
			svc := NewService(nil)
			for _, tc := range []struct {
				name string
				call func() error
			}{
				{"Shorten", func() error { _, err := svc.Shorten(ctx, ShortenRequest{}); return err }},
				{"Resolve", func() error { _, err := svc.Resolve(ctx, ""); return err }},
				{"List", func() error { _, err := svc.List(ctx, "", Page{}); return err }},
				{"Delete", func() error { return svc.Delete(ctx, "", "") }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if err := tc.call(); err != ctx.Err() {
						t.Fatalf("error = %v; want %v", err, ctx.Err())
					}
				})
			}
		})
	}
}

func TestWithFirstID(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []Option
		want uint64
	}{
		{"default", nil, 1},
		{"configured", []Option{WithFirstID(42)}, 42},
		{"zero", []Option{WithFirstID(0)}, 1},
		{"zero preserves previous option", []Option{WithFirstID(42), WithFirstID(0)}, 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(NoopStore{}, tc.opts...)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			req := ShortenRequest{Target: "https://go.dev", OwnerID: "cli"}
			if _, err := svc.Shorten(ctx, req); err != context.Canceled {
				t.Fatalf("canceled Shorten error = %v", err)
			}
			for offset := uint64(0); offset < 2; offset++ {
				got, err := svc.Shorten(context.Background(), req)
				if err != nil || got.ID != tc.want+offset {
					t.Fatalf("Shorten ID = %d, %v; want %d", got.ID, err, tc.want+offset)
				}
			}
		})
	}
}
