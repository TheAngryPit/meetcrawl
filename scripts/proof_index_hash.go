//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: proof_index_hash <meetcrawl.db>")
		os.Exit(2)
	}
	store, err := archive.Open(context.Background(), os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()
	dump, err := store.OrderedRowDump(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(source.ContentHash(dump))
}
