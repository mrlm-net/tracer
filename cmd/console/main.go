package main

import (
	"os"

	"github.com/mrlm-net/tracer/internal/console"
)

func main() {
	os.Exit(
		console.Run(os.Args[1:], os.Stdout, os.Stderr),
	)
}
