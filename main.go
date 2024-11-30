package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ghaniswara/dating-app/internal"
)

func main() {
	ctx := context.Background()
	if err := internal.Run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
