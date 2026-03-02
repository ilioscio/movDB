// Typst backend for movdb.
//
// Unlike the HTML backend, this renderer does not consume pre-computed layout
// from the layout package. Typst's typesetting engine handles column
// assignment, line-breaking, and pagination natively. The Go code only needs
// to emit entries sequentially; #block(breakable: false) ensures no entry is
// ever split across a column or page boundary.
//
// # Compiling the output
//
//	movdb -fmt typst -o list.typ /path/to/movies
//	typst compile list.typ list.pdf
//
// Or in one pipeline step via stdin:
//
//	movdb -fmt typst /path/to/movies | typst compile - list.pdf
package render

import (
	"fmt"
	"strings"

	"github.com/ilioscio/movDB/internal/parser"
)

// RenderTypst generates a Typst (.typ) source document from a sorted slice of
// entries. The caller is responsible for sorting entries before calling this
// function. Entry numbers are assigned sequentially starting at 1.
func RenderTypst(entries []parser.Entry, cfg Config) string {
	var b strings.Builder

	writeTypstPreamble(&b, cfg)

	// Open the two-column region. All entries go inside this block.
	// After the closing ] we return to single-column for the errata section.
	fmt.Fprintf(&b, "#columns(2, gutter: 0.4in)[\n")

	var errata []typstErrataItem
	for i, e := range entries {
		num := i + 1
		if cfg.ByYear {
			fmt.Fprintf(&b, "#metadata(\"%s\")<entry-mark>\n", yearLabel(e.Year))
		} else {
			fmt.Fprintf(&b, "#metadata(\"%s\")<entry-mark>\n", typstStringEscape(titlePrefix(e.DisplayTitle)))
		}
		if num%2 == 1 {
			fmt.Fprintf(&b, "#entry(shaded: true)[%d][%s]\n", num, typstEscape(e.DisplayTitle))
		} else {
			fmt.Fprintf(&b, "#entry[%d][%s]\n", num, typstEscape(e.DisplayTitle))
		}
		if len(e.Errata) > 0 {
			errata = append(errata, typstErrataItem{Number: num, Entry: e})
		}
	}

	fmt.Fprintf(&b, "]\n") // close #columns

	if len(errata) > 0 {
		writeTypstErrata(&b, errata, cfg)
	}

	return b.String()
}

// ─── internal helpers ─────────────────────────────────────────────────────────

type typstErrataItem struct {
	Number int
	Entry  parser.Entry
}

func writeTypstPreamble(b *strings.Builder, cfg Config) {
	// %%  →  literal % in fmt output (needed for Typst's 100% width).
	fmt.Fprintf(b, `#set page(
  paper: "us-letter",
  margin: (x: 0.75in, y: 0.75in),
  header: context [
    #set text(size: 12pt)
    #let markers = query(<entry-mark>).filter(m => m.location().page() == here().page())
    #let range-str = if markers.len() > 0 {
      "'" + markers.first().value + "' to '" + markers.last().value + "'"
    } else { "" }
    #grid(
      columns: (1fr, auto, 1fr),
      align: (left + horizon, center + horizon, right + horizon),
      [#strong[%s] #h(1em) %s],
      [#strong[#range-str]],
      [Page #counter(page).display()],
    )
    #v(2pt)
    #line(length: 100%%, stroke: 0.5pt)
    #v(2pt)
  ],
)
#set text(font: "Linux Libertine O", size: 10pt)
#set par(leading: 4pt, spacing: 0pt)

#let entry(num, title, shaded: false) = block(
  breakable: false,
  width: 100%%,
  fill: if shaded { luma(232) } else { none },
  inset: (x: 2pt, y: 3pt),
  grid(
    columns: (3em, 1fr),
    align: (right + top, left + top),
    column-gutter: 0.4em,
    num, title,
  )
)

`, typstEscape(cfg.Title), typstEscape(cfg.Date))
}

func writeTypstErrata(b *strings.Builder, items []typstErrataItem, cfg Config) {
	fmt.Fprintf(b, "\n#pagebreak()\n")
	fmt.Fprintf(b, "= %s — Errata\n\n", typstEscape(cfg.Title))
	fmt.Fprintf(b, "The following entries require manual attention. ")
	fmt.Fprintf(b, "They appear in the main list above but may have incorrect or missing metadata.\n\n")

	byKind := typstGroupByKind(items)

	if group, ok := byKind[parser.ErrataLeadingThe]; ok {
		fmt.Fprintf(b, "== Directories with leading \"The\" (auto-corrected in list)\n\n")
		for _, item := range group {
			writeTypstErrataItem(b, item)
		}
		fmt.Fprintf(b, "\n")
	}

	if group, ok := byKind[parser.ErrataMissingYear]; ok {
		fmt.Fprintf(b, "== Directories with missing year\n\n")
		for _, item := range group {
			writeTypstErrataItem(b, item)
		}
		fmt.Fprintf(b, "\n")
	}

	if group, ok := byKind[parser.ErrataInvalidYear]; ok {
		fmt.Fprintf(b, "== Directories with unrecognised year format\n\n")
		for _, item := range group {
			writeTypstErrataItem(b, item)
		}
		fmt.Fprintf(b, "\n")
	}
}

func writeTypstErrataItem(b *strings.Builder, item typstErrataItem) {
	// Format: - *#N* — `raw-dir-name`
	//           _issue message_
	fmt.Fprintf(b, "- *\\#%d* — `%s`",
		item.Number, typstEscapeRaw(item.Entry.RawDir))
	for _, f := range item.Entry.Errata {
		fmt.Fprintf(b, "\\ _%s_", typstEscape(f.Message))
	}
	fmt.Fprintf(b, "\n")
}

// typstGroupByKind maps each ErrataKind to the items that have that flag.
// An item with multiple flags appears under each relevant kind.
func typstGroupByKind(items []typstErrataItem) map[parser.ErrataKind][]typstErrataItem {
	m := make(map[parser.ErrataKind][]typstErrataItem)
	for _, item := range items {
		seen := make(map[parser.ErrataKind]bool)
		for _, f := range item.Entry.Errata {
			if !seen[f.Kind] {
				m[f.Kind] = append(m[f.Kind], item)
				seen[f.Kind] = true
			}
		}
	}
	return m
}

// typstEscape escapes characters that carry special meaning inside a Typst
// content block (inside [...]).  The most common hit in movie titles is '#'
// (e.g. "Alien #1 - Alien (1979)").
func typstEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '#', '*', '_', '`', '$', '@', '~', '[', ']', '<', '>':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// typstStringEscape escapes a string for use inside a Typst string literal
// ("..."). Only backslash and double-quote need escaping in that context.
func typstStringEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// typstEscapeRaw escapes the backtick delimiter used inside Typst raw spans.
// Backticks in the content would prematurely close the raw span.
func typstEscapeRaw(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}
