// Binary djinn is the CLI entry point.
// All domain logic lives in the app package (hexagonal architecture).
package main

import (
	"fmt"
	"os"

	"github.com/dpopsuev/djinn/app"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "djinn: %v\n", err)
		os.Exit(1)
	}
}
