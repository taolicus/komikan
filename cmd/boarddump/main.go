// Command boarddump prints the canonical Komikan board for review.
//
// It renders the board as an ASCII sketch, matching the diagram in
// docs/board.md, and can write an SVG rendering for visual inspection.
//
//	go run ./cmd/boarddump
//	go run ./cmd/boarddump -svg docs/board.svg
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/taolicus/komikan/internal/board"
)

func main() {
	svgPath := flag.String("svg", "", "write an SVG rendering to this path")
	flag.Parse()

	b, pos, err := board.Standard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "boarddump: %v\n", err)
		os.Exit(1)
	}
	if err := board.Validate(b, pos); err != nil {
		fmt.Fprintf(os.Stderr, "boarddump: invalid board: %v\n", err)
		os.Exit(1)
	}

	pangui, trewa := pos.Counts()
	fmt.Printf("%d nodes, %d edges, %d Pangui, %d Trewa\n\n", len(b.Nodes()), len(b.Edges()), pangui, trewa)
	fmt.Println(b.RenderASCII(pos))
	fmt.Printf("\n%s\n", b.RenderASCIIWithPieces(pos))

	if *svgPath != "" {
		// A standalone SVG file needs an XML declaration; inline in HTML it
		// does not.
		doc := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + b.RenderSVG(pos)
		if err := os.WriteFile(*svgPath, []byte(doc), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "boarddump: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nwrote %s\n", *svgPath)
	}
}
