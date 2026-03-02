package render

import (
	"strings"
	"testing"

	"github.com/ilioscio/movDB/internal/parser"
)

// ─── typstEscape ─────────────────────────────────────────────────────────────

func TestTypstEscape_PlainTitle(t *testing.T) {
	got := typstEscape("Wizard of Oz, The (1939)")
	if got != "Wizard of Oz, The (1939)" {
		t.Errorf("plain title should be unchanged, got %q", got)
	}
}

func TestTypstEscape_Hash(t *testing.T) {
	// '#' is the most common special char in movie titles ("Alien #1").
	got := typstEscape("Alien #1 - Alien (1979)")
	// The escaped form is \#1; since \#1 still contains #1 as a substring,
	// we verify the backslash-hash sequence is present rather than checking
	// for absence of '#'.
	if !strings.Contains(got, `\#1`) {
		t.Errorf("expected escaped '\\#1' in output, got %q", got)
	}
}

func TestTypstEscape_Backslash(t *testing.T) {
	got := typstEscape(`Back\slash (2001)`)
	if !strings.Contains(got, `\\`) {
		t.Errorf("backslash not doubled: %q", got)
	}
}

func TestTypstEscape_Brackets(t *testing.T) {
	got := typstEscape("Title [Extra] (2001)")
	if strings.Contains(got, "[Extra]") {
		t.Errorf("unescaped brackets in %q", got)
	}
}

func TestTypstEscape_AllSpecials(t *testing.T) {
	specials := []rune{'\\', '#', '*', '_', '`', '$', '@', '~', '[', ']', '<', '>'}
	for _, r := range specials {
		input := string([]rune{'A', r, 'Z'})
		got := typstEscape(input)
		// The escaped form must contain \<char>, not just <char>.
		want := `\` + string(r)
		if !strings.Contains(got, want) {
			t.Errorf("typstEscape(%q): expected %q in output, got %q", input, want, got)
		}
	}
}

// ─── RenderTypst structure ────────────────────────────────────────────────────

func makeTypstEntries(titles []string) []parser.Entry {
	entries := make([]parser.Entry, len(titles))
	for i, t := range titles {
		entries[i] = parser.Entry{DisplayTitle: t, SortKey: strings.ToLower(t), RawDir: t}
	}
	return entries
}

var typstCfg = Config{Title: "Movie List", Date: "2026-03-02"}

func TestRenderTypst_HasPreamble(t *testing.T) {
	out := RenderTypst(makeTypstEntries([]string{"Wall-E (2008)"}), typstCfg)
	if !strings.Contains(out, "#set page(") {
		t.Error("missing #set page declaration")
	}
	if !strings.Contains(out, `paper: "us-letter"`) {
		t.Error("missing paper size")
	}
	if !strings.Contains(out, "#let entry(") {
		t.Error("missing entry macro definition")
	}
}

func TestRenderTypst_HeaderMetadata(t *testing.T) {
	out := RenderTypst(nil, typstCfg)
	if !strings.Contains(out, "2026-03-02") {
		t.Error("date not in output")
	}
	if !strings.Contains(out, "Movie List") {
		t.Error("title not in output")
	}
	if !strings.Contains(out, "counter(page)") {
		t.Error("page counter not in output")
	}
}

func TestRenderTypst_TwoColumnBlock(t *testing.T) {
	out := RenderTypst(makeTypstEntries([]string{"Alpha (2001)"}), typstCfg)
	if !strings.Contains(out, "#columns(2") {
		t.Error("missing #columns call")
	}
}

func TestRenderTypst_EntryNumbers(t *testing.T) {
	titles := []string{"Alpha (2001)", "Bravo (2002)", "Charlie (2003)"}
	out := RenderTypst(makeTypstEntries(titles), typstCfg)
	// Odd entries: #entry(shaded: true)[N][title]
	// Even entries: #entry[N][title]
	// In both cases the number appears as [N][ — match on that.
	for _, num := range []string{"[1][", "[2][", "[3]["} {
		if !strings.Contains(out, num) {
			t.Errorf("missing entry with number pattern %q", num)
		}
	}
}

func TestRenderTypst_OddEntriesShaded(t *testing.T) {
	titles := []string{"Alpha (2001)", "Bravo (2002)", "Charlie (2003)"}
	out := RenderTypst(makeTypstEntries(titles), typstCfg)
	// Entry 1 and 3 (odd) must have shaded: true.
	if !strings.Contains(out, "entry(shaded: true)[1]") {
		t.Error("entry 1 (odd) should be shaded")
	}
	if !strings.Contains(out, "entry(shaded: true)[3]") {
		t.Error("entry 3 (odd) should be shaded")
	}
	// Entry 2 (even) must NOT have shaded: true.
	if strings.Contains(out, "entry(shaded: true)[2]") {
		t.Error("entry 2 (even) should not be shaded")
	}
}

func TestRenderTypst_EscapesHashInTitle(t *testing.T) {
	out := RenderTypst(makeTypstEntries([]string{"Alien #1 - Alien (1979)"}), typstCfg)
	// Raw '#1' must not appear unescaped after the entry number brackets.
	// The entry is emitted as: #entry[1][Alien \#1 - Alien (1979)]
	if strings.Contains(out, "[Alien #1") {
		t.Error("unescaped '#' in entry title")
	}
	if !strings.Contains(out, `\#1`) {
		t.Error("expected escaped \\#1 in entry title")
	}
}

func TestRenderTypst_EntryTitles(t *testing.T) {
	out := RenderTypst(makeTypstEntries([]string{"Citizen Kane (1941)"}), typstCfg)
	if !strings.Contains(out, "Citizen Kane (1941)") {
		t.Error("entry title not found in output")
	}
}

// ─── Errata section ───────────────────────────────────────────────────────────

func TestRenderTypst_NoErrataWhenClean(t *testing.T) {
	out := RenderTypst(makeTypstEntries([]string{"Clean Title (2001)"}), typstCfg)
	if strings.Contains(out, "pagebreak") {
		t.Error("errata pagebreak should not appear when there are no issues")
	}
}

func TestRenderTypst_ErrataSection(t *testing.T) {
	entries := []parser.Entry{
		{
			DisplayTitle: "Cartoons Misc",
			SortKey:      "cartoons misc",
			RawDir:       "Cartoons Misc",
			Errata: []parser.ErrataFlag{
				{Kind: parser.ErrataMissingYear, Message: "no year found"},
			},
		},
	}
	out := RenderTypst(entries, typstCfg)
	if !strings.Contains(out, "pagebreak") {
		t.Error("errata section should include a pagebreak")
	}
	if !strings.Contains(out, "Errata") {
		t.Error("errata heading missing")
	}
	if !strings.Contains(out, "Cartoons Misc") {
		t.Error("errata raw dir missing")
	}
}

func TestRenderTypst_ErrataGroupedByKind(t *testing.T) {
	entries := []parser.Entry{
		{
			DisplayTitle: "Foo, The (1999)",
			RawDir:       "The Foo (1999)",
			Errata:       []parser.ErrataFlag{{Kind: parser.ErrataLeadingThe, Message: "leading the"}},
		},
		{
			DisplayTitle: "Bar",
			RawDir:       "Bar",
			Errata:       []parser.ErrataFlag{{Kind: parser.ErrataMissingYear, Message: "no year"}},
		},
	}
	out := RenderTypst(entries, typstCfg)
	if !strings.Contains(out, `leading "The"`) {
		t.Error("leading-The errata heading missing")
	}
	if !strings.Contains(out, "missing year") {
		t.Error("missing-year errata heading missing")
	}
}
