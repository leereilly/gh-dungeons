package main

import (
	"fmt"
	"os"

	"github.com/leereilly/gh-dungeons/game"
)

func main() {
	// Check for --merge flag
	mergeMode := false
	for _, arg := range os.Args[1:] {
		if arg == "--merge" {
			mergeMode = true
			break
		}
	}

	g, err := game.New(game.WithMergeMode(mergeMode))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing game: %v\n", err)
		os.Exit(1)
	}
	defer g.Close()

	if err := g.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running game: %v\n", err)
		os.Exit(1)
	}
}
