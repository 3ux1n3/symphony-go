package main

import (
	"fmt"
	"os"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}

	fmt.Fprintln(os.Stderr, "symphony: Phase 0 scaffold is ready; runtime implementation is pending")
}
