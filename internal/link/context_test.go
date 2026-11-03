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
