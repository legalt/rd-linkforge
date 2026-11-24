package store

import (
	"context"
	"testing"
	"time"

	"github.com/legalt/rd-linkforge/internal/link"
)

func TestStoreContext(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	for _, ctx := range []context.Context{canceled, expired} {
		t.Run(ctx.Err().Error(), func(t *testing.T) {
			st := New()
			original := link.Link{ID: 1, Code: "kept", OwnerID: "cli"}
			if _, err := st.Create(context.Background(), original); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name string
				call func() error
			}{
				{"Create duplicate", func() error { _, err := st.Create(ctx, original); return err }},
				{"Create new", func() error { _, err := st.Create(ctx, link.Link{ID: 2, Code: "new"}); return err }},
				{"ByCode", func() error { _, err := st.ByCode(ctx, "missing"); return err }},
				{"ListByOwner", func() error { _, err := st.ListByOwner(ctx, "", link.Page{Cursor: "!"}); return err }},
				{"Delete existing", func() error { return st.Delete(ctx, 1) }},
				{"Delete missing", func() error { return st.Delete(ctx, 99) }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if err := tc.call(); err != ctx.Err() {
						t.Fatalf("error = %v; want %v", err, ctx.Err())
					}
					if got := st.All(); len(got) != 1 || got[0] != original {
						t.Fatalf("canceled call changed the store: %+v", got)
					}
				})
			}
		})
	}
}
