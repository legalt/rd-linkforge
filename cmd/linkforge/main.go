package main

import (
	"fmt"
	"os"

	"github.com/skskuzan/rd-linkforge/internal/link"
	"github.com/skskuzan/rd-linkforge/internal/store"
)

func main() {
	s := store.New()

	var id uint64 = 1

	for _, v := range os.Args[1:] {
		link, err := link.New(id, v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		if err := store.Add(s, link); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		id++
	}

	links := store.All(s)
	for _, link := range links {
		fmt.Printf("%s  %s\n", link.Code, link.Target)
	}

	fmt.Printf("\nTotal: %d\n", len(links))
}
