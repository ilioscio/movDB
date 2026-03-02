// Package layout handles word-wrapping titles and assigning entries to a
// paginated two-column layout.
//
// # Page geometry (US Letter, 0.75" margins, Courier New 9pt)
//
// The layout is computed in a character-based model. Each "row" corresponds
// to one line of monospaced text. The HTML renderer must use a font size
// that makes these character counts accurate on paper — see render package.
//
//	Column composition (per column):
//	  [number field] [sep] [title text]
//	       5     +   2   +    36      = 43 chars wide
//
//	Page rows:
//	  @page margin: 0.75" × 2 = 54pt × 2 = 108pt removed from 792pt (11")
//	  Usable height: 684pt
//	  Header area:  ~32pt (header line + spacing)
//	  Content:      652pt / 12pt per row ≈ 54 rows per column
//
// These constants are intentionally exported so the renderer can use them
// when sizing CSS to match.
package layout

import (
	"fmt"
	"strings"

	"github.com/ilioscio/movDB/internal/parser"
)

// Layout geometry constants.
const (
	NumberWidth = 5  // right-aligned number field width ("    1" … " 9999")
	SepWidth    = 2  // spaces between number and title
	TitleWidth  = 36 // characters available for title text per line
	RowsPerCol  = 54 // max rows (lines) per column per page

	// ColWidth is the total character width of one column.
	ColWidth = NumberWidth + SepWidth + TitleWidth // 43
)

// WrappedEntry is one entry ready for layout: the title has been split into
// display lines and the entry has been assigned its number in the sorted list.
type WrappedEntry struct {
	Number  int      // 1-based position in the full sorted list
	Lines   []string // wrapped title lines (first line is the "main" line)
	IsIssue bool     // true when this entry appears in the errata section
	RawDir  string   // original directory name (for errata display)
	Errata  []parser.ErrataFlag
}

// RowCount returns how many rows this entry occupies in a column.
func (e WrappedEntry) RowCount() int { return len(e.Lines) }

// Column is one column of a page, holding a slice of entries.
type Column struct {
	Entries []WrappedEntry
}

// Page is a single printed page with a left and right column.
type Page struct {
	Number int
	Left   Column
	Right  Column
}

// BuildPages takes sorted parser entries (as returned by parser.ScanDirectory),
// numbers them, wraps their titles, and distributes them into Pages.
//
// The invariant enforced here is:
//
//	An entry whose wrapped lines would straddle a column or page boundary
//	is moved entirely to the start of the next column (or page).
//	The vacated rows at the end of the previous column are left empty.
func BuildPages(entries []parser.Entry) []Page {
	// Wrap and number all entries.
	wrapped := make([]WrappedEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = WrappedEntry{
			Number:  i + 1,
			Lines:   WrapTitle(e.DisplayTitle, TitleWidth),
			IsIssue: len(e.Errata) > 0,
			RawDir:  e.RawDir,
			Errata:  e.Errata,
		}
	}

	var pages []Page
	var currentPage Page
	pageNum := 1
	rowsUsed := 0
	onLeft := true // true = filling left column, false = filling right column

	startNewPage := func() {
		currentPage.Number = pageNum
		pages = append(pages, currentPage)
		currentPage = Page{}
		pageNum++
		rowsUsed = 0
		onLeft = true
	}

	switchToRight := func() {
		onLeft = false
		rowsUsed = 0
	}

	addEntry := func(e WrappedEntry) {
		if onLeft {
			currentPage.Left.Entries = append(currentPage.Left.Entries, e)
		} else {
			currentPage.Right.Entries = append(currentPage.Right.Entries, e)
		}
		rowsUsed += e.RowCount()
	}

	for _, e := range wrapped {
		needed := e.RowCount()

		// Does this entry fit in the remaining rows of the current column?
		if rowsUsed+needed > RowsPerCol {
			if onLeft {
				// Move to right column.
				switchToRight()
			} else {
				// Right column is also full — start a new page.
				startNewPage()
			}
		}

		// After a potential column/page advance, check once more.
		// (An entry larger than RowsPerCol will always fit — it gets its own column.)
		if rowsUsed+needed > RowsPerCol && needed <= RowsPerCol {
			// This shouldn't happen after the logic above, but guard anyway.
			if onLeft {
				switchToRight()
			} else {
				startNewPage()
			}
		}

		addEntry(e)
	}

	// Flush the last partial page.
	if len(currentPage.Left.Entries) > 0 || len(currentPage.Right.Entries) > 0 {
		currentPage.Number = pageNum
		pages = append(pages, currentPage)
	}

	return pages
}

// WrapTitle splits title into lines of at most width runes, breaking only on
// whitespace. If a single word exceeds width it is placed on its own line
// (no mid-word breaks). Leading/trailing whitespace in title is trimmed.
func WrapTitle(title string, width int) []string {
	title = strings.TrimSpace(title)
	if title == "" {
		return []string{""}
	}

	words := strings.Fields(title)
	var lines []string
	current := ""

	for _, word := range words {
		switch {
		case current == "":
			current = word
		case len(current)+1+len(word) <= width:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// FormatEntryLines renders a WrappedEntry as a slice of fixed-width strings,
// each exactly ColWidth characters wide (padded with spaces). This is the
// authoritative format string used by both the plain-text and HTML renderers.
//
//	Line 0:  "    N  Title text here..."
//	Line 1+: "       Continuation text..."
func FormatEntryLines(e WrappedEntry) []string {
	prefix := fmt.Sprintf("%*d  ", NumberWidth, e.Number)        // "    1  "
	cont := strings.Repeat(" ", NumberWidth+SepWidth)            // "       "

	result := make([]string, len(e.Lines))
	result[0] = padRight(prefix+e.Lines[0], ColWidth)
	for i := 1; i < len(e.Lines); i++ {
		result[i] = padRight(cont+e.Lines[i], ColWidth)
	}
	return result
}

// padRight pads s with trailing spaces to reach length n.
// If s is already longer than n, it is returned unchanged (no truncation).
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
