package main

import (
	"context"
	"fmt"
	"os"

	"github.com/skskuzan/rd-linkforge/internal/link"
	"github.com/skskuzan/rd-linkforge/internal/store"
)

func main() {
	links := store.New()
	svc := link.NewService(links)
	ctx := context.Background()

	for _, target := range os.Args[1:] {
		_, err := svc.Shorten(ctx, link.ShortenRequest{
			Target:  target,
			OwnerID: "demo-app",
		})
		if err != nil {
			// In demo example we dont show error message.
			// Proof:
			// $ go run ./cmd/linkforge https://example.com ftp://nope https://go.dev
			// 000001  https://example.com
			// 000002  https://go.dev

			// Total: 2

			// fmt.Fprintf(os.Stderr, "%s: %v\n", target, err)
			continue
		}
	}

	for _, l := range links.All() {
		fmt.Printf("%s  %s\n", l.Code, l.Target)
	}

	fmt.Printf("\nTotal: %d\n", links.Count())
}
