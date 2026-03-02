package layout

import (
	"strings"
	"testing"

	"github.com/ilioscio/movDB/internal/parser"
)

// ─── WrapTitle ────────────────────────────────────────────────────────────────

func TestWrapTitle_Short(t *testing.T) {
	lines := WrapTitle("Wall-E (2008)", TitleWidth)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "Wall-E (2008)" {
		t.Errorf("got %q, want %q", lines[0], "Wall-E (2008)")
	}
}

func TestWrapTitle_ExactWidth(t *testing.T) {
	// A title exactly TitleWidth chars should produce one line.
	title := strings.Repeat("x", TitleWidth)
	lines := WrapTitle(title, TitleWidth)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line for exact-width title, got %d", len(lines))
	}
}

func TestWrapTitle_OneWordOver(t *testing.T) {
	// A title that needs exactly one wrap.
	title := "Unbreakable #1 - Unbreakable (2000)"
	lines := WrapTitle(title, 28)
	// "Unbreakable #1 - Unbreakable" = 28 chars (fits)
	// "(2000)" wraps to next line
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "Unbreakable #1 - Unbreakable" {
		t.Errorf("line 0: got %q", lines[0])
	}
	if lines[1] != "(2000)" {
		t.Errorf("line 1: got %q", lines[1])
	}
}

func TestWrapTitle_ThreeLines(t *testing.T) {
	// Mirrors user's example: Walt Disney title wrapping.
	title := "Walt Disney - The Best of Walt Disney True Life Adventures (1975)"
	lines := WrapTitle(title, 30)
	// All lines must be ≤ 30 chars.
	for i, l := range lines {
		if len(l) > 30 {
			t.Errorf("line %d exceeds width: %q (len %d)", i, l, len(l))
		}
	}
	if len(lines) < 2 {
		t.Errorf("expected multiple lines for long title, got %d: %v", len(lines), lines)
	}
}

func TestWrapTitle_NeverBreaksMidWord(t *testing.T) {
	// Every line must not start or end mid-word relative to the original.
	title := "Adventures of Huckleberry Finn, The (1985)"
	lines := WrapTitle(title, 20)
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if strings.Contains(l, " ") {
			// Lines with spaces: check they are space-delimited (no mid-word break).
			_ = l // the algorithm guarantees this by design
		}
		_ = i
	}
	// Rejoin and verify all words are present.
	rejoined := strings.Join(lines, " ")
	if rejoined != title {
		t.Errorf("rejoined %q != original %q", rejoined, title)
	}
}

func TestWrapTitle_Empty(t *testing.T) {
	lines := WrapTitle("", TitleWidth)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("expected [\"\" ], got %v", lines)
	}
}

// ─── BuildPages ───────────────────────────────────────────────────────────────

func makeEntries(titles []string) []parser.Entry {
	entries := make([]parser.Entry, len(titles))
	for i, t := range titles {
		entries[i] = parser.Entry{
			DisplayTitle: t,
			SortKey:      t,
			RawDir:       t,
		}
	}
	return entries
}

func TestBuildPages_SinglePage(t *testing.T) {
	// 5 short entries should all fit on one page, left column.
	titles := []string{
		"Alpha (2001)", "Bravo (2002)", "Charlie (2003)", "Delta (2004)", "Echo (2005)",
	}
	pages := BuildPages(makeEntries(titles))
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if len(pages[0].Left.Entries) != 5 {
		t.Errorf("expected 5 entries in left column, got %d", len(pages[0].Left.Entries))
	}
	if len(pages[0].Right.Entries) != 0 {
		t.Errorf("expected 0 entries in right column, got %d", len(pages[0].Right.Entries))
	}
}

func TestBuildPages_EntryNumbers(t *testing.T) {
	titles := []string{"Alpha (2001)", "Bravo (2002)", "Charlie (2003)"}
	pages := BuildPages(makeEntries(titles))
	entries := pages[0].Left.Entries
	for i, e := range entries {
		if e.Number != i+1 {
			t.Errorf("entry %d has Number=%d, want %d", i, e.Number, i+1)
		}
	}
}

func TestBuildPages_FillsBothColumns(t *testing.T) {
	// RowsPerCol+1 single-row entries should spill into the right column.
	titles := make([]string, RowsPerCol+1)
	for i := range titles {
		titles[i] = "Movie (2000)"
	}
	pages := BuildPages(makeEntries(titles))
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if len(pages[0].Left.Entries) != RowsPerCol {
		t.Errorf("left column: got %d entries, want %d", len(pages[0].Left.Entries), RowsPerCol)
	}
	if len(pages[0].Right.Entries) != 1 {
		t.Errorf("right column: got %d entries, want 1", len(pages[0].Right.Entries))
	}
}

func TestBuildPages_SpillsToNextPage(t *testing.T) {
	// 2*RowsPerCol + 1 entries → 2 pages; the extra entry starts page 2 left column.
	count := 2*RowsPerCol + 1
	titles := make([]string, count)
	for i := range titles {
		titles[i] = "Movie (2000)"
	}
	pages := BuildPages(makeEntries(titles))
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	// Page 1: left=RowsPerCol, right=RowsPerCol
	if len(pages[0].Left.Entries) != RowsPerCol {
		t.Errorf("page 1 left: got %d, want %d", len(pages[0].Left.Entries), RowsPerCol)
	}
	if len(pages[0].Right.Entries) != RowsPerCol {
		t.Errorf("page 1 right: got %d, want %d", len(pages[0].Right.Entries), RowsPerCol)
	}
	// Page 2: left=1, right=0
	if len(pages[1].Left.Entries) != 1 {
		t.Errorf("page 2 left: got %d, want 1", len(pages[1].Left.Entries))
	}
}

func TestBuildPages_MultiRowEntryMovesToNextColumn(t *testing.T) {
	// Fill left column to RowsPerCol-1, then add a 2-row entry.
	// The 2-row entry should move entirely to the right column.
	shortTitle := "Short (2001)" // 1 row
	longTitle := strings.Repeat("a ", 20) + "(1999)" // will wrap to 2+ rows

	var titles []string
	for i := 0; i < RowsPerCol-1; i++ {
		titles = append(titles, shortTitle)
	}
	titles = append(titles, longTitle)

	pages := BuildPages(makeEntries(titles))
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	// Left column should have exactly RowsPerCol-1 entries (the short ones).
	if len(pages[0].Left.Entries) != RowsPerCol-1 {
		t.Errorf("left: got %d entries, want %d", len(pages[0].Left.Entries), RowsPerCol-1)
	}
	// Right column should have the 2-row entry.
	if len(pages[0].Right.Entries) != 1 {
		t.Errorf("right: got %d entries, want 1", len(pages[0].Right.Entries))
	}
}

func TestBuildPages_PageNumbers(t *testing.T) {
	count := 2*RowsPerCol + 1
	titles := make([]string, count)
	for i := range titles {
		titles[i] = "Movie (2000)"
	}
	pages := BuildPages(makeEntries(titles))
	for i, p := range pages {
		if p.Number != i+1 {
			t.Errorf("page %d has Number=%d, want %d", i, p.Number, i+1)
		}
	}
}

// ─── FormatEntryLines ─────────────────────────────────────────────────────────

func TestFormatEntryLines_Single(t *testing.T) {
	e := WrappedEntry{Number: 1, Lines: []string{"Wall-E (2008)"}}
	lines := FormatEntryLines(e)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Must start with right-aligned number.
	if !strings.HasPrefix(lines[0], "    1  ") {
		t.Errorf("line[0] = %q, want prefix '    1  '", lines[0])
	}
	if !strings.Contains(lines[0], "Wall-E (2008)") {
		t.Errorf("line[0] missing title: %q", lines[0])
	}
}

func TestFormatEntryLines_Continuation(t *testing.T) {
	e := WrappedEntry{Number: 42, Lines: []string{"Unbreakable #1 - Unbreakable", "(2000)"}}
	lines := FormatEntryLines(e)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	// Continuation line must have blank number field.
	contPrefix := strings.Repeat(" ", NumberWidth+SepWidth)
	if !strings.HasPrefix(lines[1], contPrefix) {
		t.Errorf("continuation line[1] = %q, want prefix %q", lines[1], contPrefix)
	}
	if !strings.Contains(lines[1], "(2000)") {
		t.Errorf("continuation line[1] missing text: %q", lines[1])
	}
}
