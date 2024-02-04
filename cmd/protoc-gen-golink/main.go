package main

import (
	"fmt"
	"github.com/stalomeow/protolinker/internal/app"
	"os"
	"path/filepath"
)

func main() {
	g := app.NewGoGenerator("1.0.0")

	if err := app.Run(g.Execute); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}
