package main

import (
	"fmt"
	"os"

	"github.com/leereilly/gh-dungeons/game"
)

func main() {
	g, err := game.New()
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
