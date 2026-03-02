package render

import (
	"strings"
	"testing"

	"github.com/ilioscio/movDB/internal/layout"
	"github.com/ilioscio/movDB/internal/parser"
)

func singlePage(entries []layout.WrappedEntry) []layout.Page {
	return []layout.Page{
		{
			Number: 1,
			Left:   layout.Column{Entries: entries},
		},
	}
}

var testCfg = Config{Title: "Movie List", Date: "2026-01-15"}

// ─── Basic structure ──────────────────────────────────────────────────────────

func TestRenderHTML_IsValidHTML(t *testing.T) {
	pages := singlePage([]layout.WrappedEntry{
		{Number: 1, Lines: []string{"Wall-E (2008)"}},
	})
	out := RenderHTML(pages, nil, testCfg)

	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Error("output does not start with <!DOCTYPE html>")
	}
	if !strings.Contains(out, "</html>") {
		t.Error("output is missing closing </html>")
	}
}

func TestRenderHTML_PageHeader(t *testing.T) {
	pages := singlePage(nil)
	out := RenderHTML(pages, nil, testCfg)

	if !strings.Contains(out, "2026-01-15") {
		t.Error("output missing date")
	}
	if !strings.Contains(out, "Movie List") {
		t.Error("output missing title")
	}
	if !strings.Contains(out, "Page 1") {
		t.Error("output missing page number")
	}
}

func TestRenderHTML_MultiplePages(t *testing.T) {
	pages := []layout.Page{
		{Number: 1, Left: layout.Column{Entries: []layout.WrappedEntry{
			{Number: 1, Lines: []string{"Alpha (2001)"}},
		}}},
		{Number: 2, Left: layout.Column{Entries: []layout.WrappedEntry{
			{Number: 2, Lines: []string{"Bravo (2002)"}},
		}}},
	}
	out := RenderHTML(pages, nil, testCfg)

	if !strings.Contains(out, "Page 1") {
		t.Error("missing Page 1 header")
	}
	if !strings.Contains(out, "Page 2") {
		t.Error("missing Page 2 header")
	}
}

// ─── Entry rendering ──────────────────────────────────────────────────────────

func TestRenderHTML_EntryContent(t *testing.T) {
	pages := singlePage([]layout.WrappedEntry{
		{Number: 42, Lines: []string{"Citizen Kane (1941)"}},
	})
	out := RenderHTML(pages, nil, testCfg)

	if !strings.Contains(out, "Citizen Kane (1941)") {
		t.Error("entry title missing from output")
	}
	// Number should appear somewhere in the entry lines.
	if !strings.Contains(out, "42") {
		t.Error("entry number missing from output")
	}
}

func TestRenderHTML_WrappedEntryHasMultipleLines(t *testing.T) {
	pages := singlePage([]layout.WrappedEntry{
		{Number: 1, Lines: []string{"100-Year-Old Man Who Climbed Out", "the Window (2013)"}},
	})
	out := RenderHTML(pages, nil, testCfg)

	if !strings.Contains(out, "100-Year-Old Man Who Climbed Out") {
		t.Error("first wrapped line missing")
	}
	if !strings.Contains(out, "the Window (2013)") {
		t.Error("continuation line missing")
	}
}

// ─── Errata section ───────────────────────────────────────────────────────────

func TestRenderHTML_NoErrataWhenClean(t *testing.T) {
	pages := singlePage(nil)
	out := RenderHTML(pages, nil, testCfg)

	// The errata section is a <div class="errata-page"> — check it's absent.
	// (CSS rules contain "errata" in class selectors, so we check for the div.)
	if strings.Contains(out, `class="errata-page"`) {
		t.Error("errata-page div should not appear when there are no issues")
	}
}

func TestRenderHTML_ErrataAppears(t *testing.T) {
	errata := []layout.WrappedEntry{
		{
			Number:  7,
			Lines:   []string{"Cartoons Misc"},
			IsIssue: true,
			RawDir:  "Cartoons Misc",
			Errata: []parser.ErrataFlag{
				{Kind: parser.ErrataMissingYear, Message: "no year found"},
			},
		},
	}
	pages := singlePage(nil)
	out := RenderHTML(pages, errata, testCfg)

	if !strings.Contains(out, "Errata") {
		t.Error("errata section heading missing")
	}
	if !strings.Contains(out, "Cartoons Misc") {
		t.Error("errata entry raw dir missing")
	}
	if !strings.Contains(out, "no year found") {
		t.Error("errata issue message missing")
	}
}

func TestRenderHTML_ErrataGroupedByKind(t *testing.T) {
	errata := []layout.WrappedEntry{
		{
			Number: 1, Lines: []string{"Foo (1999)"}, IsIssue: true, RawDir: "The Foo (1999)",
			Errata: []parser.ErrataFlag{{Kind: parser.ErrataLeadingThe, Message: "leading the"}},
		},
		{
			Number: 2, Lines: []string{"Bar"}, IsIssue: true, RawDir: "Bar",
			Errata: []parser.ErrataFlag{{Kind: parser.ErrataMissingYear, Message: "no year"}},
		},
	}
	pages := singlePage(nil)
	out := RenderHTML(pages, errata, testCfg)

	// Both errata kinds should appear.
	if !strings.Contains(out, "leading") {
		t.Error("leading-The errata group missing")
	}
	if !strings.Contains(out, "missing year") {
		t.Error("missing-year errata group heading missing")
	}
}

// ─── HTML escaping ────────────────────────────────────────────────────────────

func TestRenderHTML_EscapesSpecialChars(t *testing.T) {
	pages := singlePage([]layout.WrappedEntry{
		{Number: 1, Lines: []string{"It's a Wonderful Life (1946)"}},
	})
	out := RenderHTML(pages, nil, Config{Title: "List & More", Date: "2026-01-15"})

	// The ampersand in the title must be HTML-escaped.
	if strings.Contains(out, "<title>List & More</title>") {
		t.Error("ampersand in title was not escaped")
	}
	if !strings.Contains(out, "List &amp; More") {
		t.Error("expected escaped ampersand in output")
	}
}
