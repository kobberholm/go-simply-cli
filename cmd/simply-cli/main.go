package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kobberholm/go-simply-cli/internal/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
