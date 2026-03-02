// Package render generates a print-ready HTML document from a pre-computed
// page layout.
//
// # Design notes for potential Typst migration
//
// The layout is fully pre-computed by the layout package before this renderer
// is called. The renderer is a pure "data → markup" transformation with no
// pagination logic of its own. This separation means the layout package can
// be reused with a Typst backend by simply implementing a different Render
// function that emits .typ markup instead of HTML.
//
// Key mapping from layout constants to CSS:
//
//	layout.ColWidth  (43 chars)  →  CSS width: 43ch  on each .column div
//	layout.TitleWidth (36 chars)  →  CSS max-width of .title-text
//	Font: Courier New 9pt         →  1ch ≈ 5.4pt → 43ch ≈ 3.23" per column
//	Line height: 12pt             →  layout.RowsPerCol (54) × 12pt = 648pt usable
//	                                 which fits in 9.5" page height after header
package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/ilioscio/movDB/internal/layout"
	"github.com/ilioscio/movDB/internal/parser"
)

// Config holds document-level metadata used in page headers and the HTML title.
type Config struct {
	Title  string // e.g. "Movie List"
	Date   string // e.g. "2026-01-15"
	ByYear bool   // true when entries are sorted by year; affects page range display
}

// RenderHTML produces a complete, self-contained HTML document.
// pages is the pre-computed layout; errataEntries is the subset of entries
// that had parsing issues (they also appear in the main list).
func RenderHTML(pages []layout.Page, errataEntries []layout.WrappedEntry, cfg Config) string {
	var b strings.Builder

	writeHeader(&b, cfg)

	for _, p := range pages {
		writePage(&b, p, cfg)
	}

	if len(errataEntries) > 0 {
		writeErrata(&b, errataEntries, cfg)
	}

	writeFooter(&b)

	return b.String()
}

// ─── page sections ────────────────────────────────────────────────────────────

func writeHeader(b *strings.Builder, cfg Config) {
	fmt.Fprintf(b, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
/* ── Print geometry ──────────────────────────────────────────────────────────
   US Letter: 8.5" × 11" with 0.75" margins → 7" × 9.5" usable.
   Courier New at 9pt: 1ch ≈ 5.4pt.
   Two columns of 43ch each with a 2ch gap = 88ch ≈ 6.6" (fits in 7").
   Line height 12pt × 54 rows = 648pt ≈ 9" (fits below the 32pt header area).
   ─────────────────────────────────────────────────────────────────────────── */

@page {
  size: letter portrait;
  margin: 0.75in;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: 'Courier New', Courier, monospace;
  font-size: 9pt;
  line-height: 12pt;
  background: #888;
}

/* ── Screen preview: show each .page as a physical letter sheet ───────────── */
@media screen {
  body {
    padding: 1rem;
  }
  .page, .errata-page {
    width: 7in;
    min-height: 9.5in;
    background: white;
    margin: 0.5in auto;
    padding: 0;
    box-shadow: 0 2px 12px rgba(0,0,0,0.4);
  }
}

/* ── Print: each .page div = exactly one sheet ───────────────────────────── */
@media print {
  body {
    background: white;
  }
  .page, .errata-page {
    page-break-after: always;
    break-after: page;
  }
  .page:last-of-type, .errata-page:last-of-type {
    page-break-after: avoid;
    break-after: avoid;
  }
}

/* ── Page header ─────────────────────────────────────────────────────────── */
/* grid: 3 cells — outer edges hold page# and title+date (1fr each so range
   stays truly centered), auto center holds the range.
   Odd pages  (recto): [title date] | [range] | [page#]
   Even pages (verso): [page#]      | [range] | [title date]
   This keeps page# at the outer edge for easy thumb-through on duplex prints. */
.page-header {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: baseline;
  border-bottom: 1px solid #000;
  padding-bottom: 3pt;
  margin-bottom: 6pt;
  white-space: nowrap;
  font-size: 11pt;
}
.header-left   { display: flex; gap: 1.5em; }
.header-right  { display: flex; gap: 1.5em; justify-content: flex-end; }
.header-title  { font-weight: bold; }
.header-date   {}
.header-range  { text-align: center; padding: 0 1em; font-weight: bold; }
.header-pagenum{ text-align: right; }

/* ── Two-column body ─────────────────────────────────────────────────────── */
.page-columns {
  display: flex;
  gap: 2ch;
}
.column {
  width: 43ch;
  overflow: hidden; /* prevent any single overlong word from breaking layout */
}

/* ── Entries ─────────────────────────────────────────────────────────────── */
.entry {
  /* Entries must never be split by the browser's reflow.
     In screen mode this prevents orphaned continuation lines. */
  break-inside: avoid;
}
.entry.shaded {
  background-color: #e8e8e8;
}
.entry-line {
  display: block;
  white-space: pre;      /* respect the pre-formatted spaces */
  overflow: hidden;      /* if a word exceeds ColWidth, clip rather than reflow */
  text-overflow: clip;
}

/* ── Errata section ──────────────────────────────────────────────────────── */
.errata-page {
  /* same visual treatment as a regular page */
}
.errata-header {
  border-bottom: 1px solid #000;
  padding-bottom: 3pt;
  margin-bottom: 6pt;
}
.errata-header h2 {
  font-size: 9pt;
  font-weight: bold;
  font-family: 'Courier New', Courier, monospace;
}
.errata-intro {
  margin-bottom: 8pt;
  font-size: 8pt;
}
.errata-group {
  margin-bottom: 12pt;
}
.errata-group h3 {
  font-size: 9pt;
  font-weight: bold;
  font-family: 'Courier New', Courier, monospace;
  text-decoration: underline;
  margin-bottom: 4pt;
}
.errata-item {
  margin-bottom: 2pt;
  font-size: 8.5pt;
}
.errata-item .entry-num {
  font-weight: bold;
}
.errata-item .raw-dir {
  font-style: italic;
  color: #444;
}
.errata-item .issue-msg {
  color: #600;
}
</style>
</head>
<body>
`, html.EscapeString(cfg.Title))
}

func writeRangeSpan(b *strings.Builder, indent, first, last string) {
	if first != "" {
		fmt.Fprintf(b, "%s<span class=\"header-range\">'%s' to '%s'</span>\n",
			indent, html.EscapeString(first), html.EscapeString(last))
	} else {
		fmt.Fprintf(b, "%s<span class=\"header-range\"></span>\n", indent)
	}
}

func writePage(b *strings.Builder, p layout.Page, cfg Config) {
	first, last := pageRangeParts(p, cfg.ByYear)

	fmt.Fprintf(b, "<div class=\"page\">\n")
	fmt.Fprintf(b, "  <div class=\"page-header\">\n")

	if p.Number%2 == 0 {
		// Even page (verso): page# | range | title+date
		fmt.Fprintf(b, "    <span>Page %d</span>\n", p.Number)
		writeRangeSpan(b, "    ", first, last)
		fmt.Fprintf(b, "    <div class=\"header-right\">\n")
		fmt.Fprintf(b, "      <span class=\"header-title\">%s</span>\n", html.EscapeString(cfg.Title))
		fmt.Fprintf(b, "      <span class=\"header-date\">%s</span>\n", html.EscapeString(cfg.Date))
		fmt.Fprintf(b, "    </div>\n")
	} else {
		// Odd page (recto): title+date | range | page#
		fmt.Fprintf(b, "    <div class=\"header-left\">\n")
		fmt.Fprintf(b, "      <span class=\"header-title\">%s</span>\n", html.EscapeString(cfg.Title))
		fmt.Fprintf(b, "      <span class=\"header-date\">%s</span>\n", html.EscapeString(cfg.Date))
		fmt.Fprintf(b, "    </div>\n")
		writeRangeSpan(b, "    ", first, last)
		fmt.Fprintf(b, "    <span class=\"header-pagenum\">Page %d</span>\n", p.Number)
	}

	fmt.Fprintf(b, "  </div>\n")

	fmt.Fprintf(b, "  <div class=\"page-columns\">\n")
	writeColumn(b, p.Left)
	writeColumn(b, p.Right)
	fmt.Fprintf(b, "  </div>\n")

	fmt.Fprintf(b, "</div>\n\n")
}

func writeColumn(b *strings.Builder, col layout.Column) {
	fmt.Fprintf(b, "    <div class=\"column\">\n")
	for _, e := range col.Entries {
		writeEntry(b, e)
	}
	fmt.Fprintf(b, "    </div>\n")
}

func writeEntry(b *strings.Builder, e layout.WrappedEntry) {
	class := "entry"
	if e.Number%2 == 1 {
		class += " shaded"
	}
	if e.IsIssue {
		class += " issue"
	}
	fmt.Fprintf(b, "      <div class=\"%s\">\n", class)

	lines := layout.FormatEntryLines(e)
	for _, l := range lines {
		fmt.Fprintf(b, "        <span class=\"entry-line\">%s</span>\n", html.EscapeString(l))
	}

	fmt.Fprintf(b, "      </div>\n")
}

func writeErrata(b *strings.Builder, entries []layout.WrappedEntry, cfg Config) {
	fmt.Fprintf(b, "<div class=\"errata-page\">\n")
	fmt.Fprintf(b, "  <div class=\"errata-header\">\n")
	fmt.Fprintf(b, "    <div class=\"page-header\">\n")
	fmt.Fprintf(b, "      <div class=\"header-left\">\n")
	fmt.Fprintf(b, "        <span class=\"header-title\">%s — Errata</span>\n", html.EscapeString(cfg.Title))
	fmt.Fprintf(b, "        <span class=\"header-date\">%s</span>\n", html.EscapeString(cfg.Date))
	fmt.Fprintf(b, "      </div>\n")
	writeRangeSpan(b, "      ", "", "")
	fmt.Fprintf(b, "      <span class=\"header-pagenum\"></span>\n")
	fmt.Fprintf(b, "    </div>\n")
	fmt.Fprintf(b, "  </div>\n")

	fmt.Fprintf(b, "  <p class=\"errata-intro\">The following entries require manual attention. ")
	fmt.Fprintf(b, "They appear in the main list above but may have incorrect or missing metadata.</p>\n\n")

	// Group by errata kind.
	byKind := groupByKind(entries)

	if items, ok := byKind[parser.ErrataLeadingThe]; ok {
		fmt.Fprintf(b, "  <div class=\"errata-group\">\n")
		fmt.Fprintf(b, "    <h3>Directories with leading &#8220;The&#8221; (auto-corrected in list)</h3>\n")
		for _, item := range items {
			writeErrataItem(b, item)
		}
		fmt.Fprintf(b, "  </div>\n\n")
	}

	if items, ok := byKind[parser.ErrataMissingYear]; ok {
		fmt.Fprintf(b, "  <div class=\"errata-group\">\n")
		fmt.Fprintf(b, "    <h3>Directories with missing year</h3>\n")
		for _, item := range items {
			writeErrataItem(b, item)
		}
		fmt.Fprintf(b, "  </div>\n\n")
	}

	if items, ok := byKind[parser.ErrataInvalidYear]; ok {
		fmt.Fprintf(b, "  <div class=\"errata-group\">\n")
		fmt.Fprintf(b, "    <h3>Directories with unrecognised year format</h3>\n")
		for _, item := range items {
			writeErrataItem(b, item)
		}
		fmt.Fprintf(b, "  </div>\n\n")
	}

	fmt.Fprintf(b, "</div>\n")
}

func writeErrataItem(b *strings.Builder, e layout.WrappedEntry) {
	fmt.Fprintf(b, "  <div class=\"errata-item\">\n")
	fmt.Fprintf(b, "    <span class=\"entry-num\">#%d</span> — ", e.Number)
	fmt.Fprintf(b, "<span class=\"raw-dir\">%s</span><br>\n", html.EscapeString(e.RawDir))
	for _, f := range e.Errata {
		fmt.Fprintf(b, "    <span class=\"issue-msg\">%s</span><br>\n", html.EscapeString(f.Message))
	}
	fmt.Fprintf(b, "  </div>\n")
}

func writeFooter(b *strings.Builder) {
	fmt.Fprintf(b, "</body>\n</html>\n")
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// pageRangeParts returns the (first, last) range strings for a page header.
// When byYear is true it uses the entry year; otherwise the title first word.
// Returns ("", "") when the page has no entries.
func pageRangeParts(p layout.Page, byYear bool) (first, last string) {
	var firstEntry, lastEntry *layout.WrappedEntry
	if len(p.Left.Entries) > 0 {
		firstEntry = &p.Left.Entries[0]
	} else if len(p.Right.Entries) > 0 {
		firstEntry = &p.Right.Entries[0]
	}
	if len(p.Right.Entries) > 0 {
		lastEntry = &p.Right.Entries[len(p.Right.Entries)-1]
	} else if len(p.Left.Entries) > 0 {
		lastEntry = &p.Left.Entries[len(p.Left.Entries)-1]
	}
	if firstEntry == nil || lastEntry == nil {
		return "", ""
	}
	if byYear {
		return yearLabel(firstEntry.Year), yearLabel(lastEntry.Year)
	}
	return titlePrefix(firstEntry.Lines[0]), titlePrefix(lastEntry.Lines[0])
}

// yearLabel formats a year for a page range header. Zero means no year found.
func yearLabel(y uint16) string {
	if y == 0 {
		return "????"
	}
	return fmt.Sprintf("%d", y)
}

// titlePrefix returns the first whitespace-separated word of s with trailing
// commas and periods stripped. Used to build page range labels like
// 'Dark' to 'Dr' (rather than 'Mask,' or 'Dr.').
func titlePrefix(s string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	return strings.TrimRight(words[0], ",.")
}

// groupByKind returns a map from ErrataKind to the entries that have that kind.
// An entry with multiple errata flags appears under each relevant kind.
func groupByKind(entries []layout.WrappedEntry) map[parser.ErrataKind][]layout.WrappedEntry {
	m := make(map[parser.ErrataKind][]layout.WrappedEntry)
	for _, e := range entries {
		seen := make(map[parser.ErrataKind]bool)
		for _, f := range e.Errata {
			if !seen[f.Kind] {
				m[f.Kind] = append(m[f.Kind], e)
				seen[f.Kind] = true
			}
		}
	}
	return m
}
