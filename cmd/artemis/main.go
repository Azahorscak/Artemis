package main

import (
	"fmt"
	"os"
)

// version is set via ldflags at build time.
var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "artemis %s\n", version)
	fmt.Fprintln(os.Stderr, "not yet implemented")
	os.Exit(1)
}
