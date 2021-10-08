package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cavaliercoder/grab/grabui"
)

func main() {
	// validate command args
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s url...\n", os.Args[0])
		os.Exit(1)
	}
	urls := os.Args[1:]

	// download files
	respch, err := grabui.GetBatch(context.Background(), 0, ".", urls...)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	// return the number of failed downloads as exit code
	failed := 0
	for resp := range respch {
		if resp.Err() != nil {
			failed++
		}
	}
	os.Exit(failed)
}
