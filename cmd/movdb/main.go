// movdb — movie directory list generator
//
// Usage:
//
//	movdb [flags] <movies-directory>
//
// Flags:
//
//	-o <file>     Output file path (default: stdout)
//	-title <str>  Document title   (default: "Movie List")
//	-date  <str>  Date string for page headers (default: today, YYYY-MM-DD)
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilioscio/movDB/internal/layout"
	"github.com/ilioscio/movDB/internal/parser"
	"github.com/ilioscio/movDB/internal/render"
)

func main() {
	title := flag.String("title", "Movie List", "Document title shown in page headers")
	output := flag.String("o", "", "Output file path (default: stdout)")
	date := flag.String("date", "", "Date string for page headers (default: today)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: movdb [flags] <movies-directory>\n\n")
		fmt.Fprintf(os.Stderr, "Reads subdirectory names from <movies-directory>, interprets them as\n")
		fmt.Fprintf(os.Stderr, "movie titles, and generates a paginated two-column HTML list.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if *date == "" {
		*date = time.Now().Format("2006-01-02")
	}

	movieDir := flag.Arg(0)

	// ── Parse ────────────────────────────────────────────────────────────────
	entries, err := parser.ScanDirectory(movieDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "movdb: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Fprintf(os.Stderr, "movdb: no movie directories found in %q\n", movieDir)
		os.Exit(1)
	}

	// ── Layout ───────────────────────────────────────────────────────────────
	pages := layout.BuildPages(entries)

	// Collect errata entries (appear in main list AND errata section).
	var errataEntries []layout.WrappedEntry
	seen := make(map[int]bool)
	for _, p := range pages {
		for _, col := range []layout.Column{p.Left, p.Right} {
			for _, e := range col.Entries {
				if e.IsIssue && !seen[e.Number] {
					errataEntries = append(errataEntries, e)
					seen[e.Number] = true
				}
			}
		}
	}

	// ── Render ───────────────────────────────────────────────────────────────
	htmlDoc := render.RenderHTML(pages, errataEntries, render.Config{
		Title: *title,
		Date:  *date,
	})

	// ── Output ───────────────────────────────────────────────────────────────
	if *output == "" {
		fmt.Print(htmlDoc)
		return
	}

	if err := os.WriteFile(*output, []byte(htmlDoc), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "movdb: writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "movdb: wrote %d entries across %d pages → %s\n",
		len(entries), len(pages), *output)
}
