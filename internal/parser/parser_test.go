package parser

import (
	"testing"
)

// ─── allDigits ────────────────────────────────────────────────────────────────

func TestAllDigits(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1999", true},
		{"0000", true},
		{"199a", false},
		{"", false},
		{"   ", false},
	}
	for _, tc := range cases {
		if got := allDigits(tc.in); got != tc.want {
			t.Errorf("allDigits(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ─── isYearContent ────────────────────────────────────────────────────────────

func TestIsYearContent(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1999", true},
		{"2005-2007", true},
		{"2010...", true},
		{"abc", false},
		{"19", false},
		{"1999x", false},
		{"1999-200", false},   // year range too short
		{"1999-20070", false}, // year range too long
		{"", false},
	}
	for _, tc := range cases {
		if got := isYearContent(tc.in); got != tc.want {
			t.Errorf("isYearContent(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ─── findYear ─────────────────────────────────────────────────────────────────

func TestFindYear(t *testing.T) {
	cases := []struct {
		in      string
		wantErr error
		wantStr string // s[span.start : span.end+1]
	}{
		{"Some Movie (1999)", nil, "(1999)"},
		{"TV Series (2005-2007)", nil, "(2005-2007)"},
		{"Ongoing Show (2010...)", nil, "(2010...)"},
		{"No Year Here", errMissingYear, ""},
		{"Has (Misc.) stuff", errInvalidYear, ""},
		// Last valid year wins when multiple parens are present.
		{"Alien #5 - Alien vs Predator ((2004))", nil, "(2004)"},
		// Extra spaces before year are fine — we pick the last valid paren.
		{"20,000 Leagues Under the Sea  (1954)", nil, "(1954)"},
	}
	for _, tc := range cases {
		span, err := findYear(tc.in)
		if err != tc.wantErr {
			t.Errorf("findYear(%q) error = %v, want %v", tc.in, err, tc.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		got := tc.in[span.start : span.end+1]
		if got != tc.wantStr {
			t.Errorf("findYear(%q) span = %q, want %q", tc.in, got, tc.wantStr)
		}
	}
}

// ─── normalizeThe ─────────────────────────────────────────────────────────────

func TestNormalizeThe(t *testing.T) {
	cases := []struct {
		in         string
		wantOut    string
		wantChanged bool
	}{
		// Standard leading-The conversion.
		{"The Wizard of Oz (1939)", "Wizard of Oz, The (1939)", true},
		{"The 39 Steps (1935)", "39 Steps, The (1935)", true},
		// Already in trailing-The form — no change.
		{"Wizard of Oz, The (1939)", "Wizard of Oz, The (1939)", false},
		// Does not start with "The ".
		{"There Will Be Blood (2007)", "There Will Be Blood (2007)", false},
		// "The" with no following space.
		{"Theorem (1968)", "Theorem (1968)", false},
	}
	for _, tc := range cases {
		out, changed := normalizeThe(tc.in)
		if out != tc.wantOut || changed != tc.wantChanged {
			t.Errorf("normalizeThe(%q) = (%q, %v), want (%q, %v)",
				tc.in, out, changed, tc.wantOut, tc.wantChanged)
		}
	}
}

// ─── ParseDirName ─────────────────────────────────────────────────────────────

func TestParseDirName_Normal(t *testing.T) {
	cases := []struct {
		in           string
		wantTitle    string
		wantErrataLen int
	}{
		{
			"Wizard of Oz, The (1939)",
			"Wizard of Oz, The (1939)",
			0,
		},
		{
			"12 Angry Men (1957)",
			"12 Angry Men (1957)",
			0,
		},
		{
			"TV Series (2005-2007)",
			"TV Series (2005-2007)",
			0,
		},
		{
			"Ongoing Show (2010...)",
			"Ongoing Show (2010...)",
			0,
		},
		// Trailing whitespace before year is stripped.
		{
			"20,000 Leagues Under the Sea  (1954)",
			"20,000 Leagues Under the Sea (1954)",
			0,
		},
	}
	for _, tc := range cases {
		e := ParseDirName(tc.in)
		if e.DisplayTitle != tc.wantTitle {
			t.Errorf("ParseDirName(%q).DisplayTitle = %q, want %q", tc.in, e.DisplayTitle, tc.wantTitle)
		}
		if len(e.Errata) != tc.wantErrataLen {
			t.Errorf("ParseDirName(%q) errata count = %d, want %d (errata: %v)",
				tc.in, len(e.Errata), tc.wantErrataLen, e.Errata)
		}
	}
}

func TestParseDirName_LeadingThe(t *testing.T) {
	e := ParseDirName("The Wizard of Oz (1939)")
	if e.DisplayTitle != "Wizard of Oz, The (1939)" {
		t.Errorf("got DisplayTitle %q, want %q", e.DisplayTitle, "Wizard of Oz, The (1939)")
	}
	if len(e.Errata) != 1 || e.Errata[0].Kind != ErrataLeadingThe {
		t.Errorf("expected exactly one ErrataLeadingThe flag, got %v", e.Errata)
	}
}

func TestParseDirName_MissingYear(t *testing.T) {
	e := ParseDirName("Cartoons Misc")
	if len(e.Errata) != 1 || e.Errata[0].Kind != ErrataMissingYear {
		t.Errorf("expected ErrataMissingYear, got %v", e.Errata)
	}
	if e.DisplayTitle != "Cartoons Misc" {
		t.Errorf("got DisplayTitle %q, want %q", e.DisplayTitle, "Cartoons Misc")
	}
}

func TestParseDirName_LeadingStar(t *testing.T) {
	// Exactly 3 asterisks → flagged.
	e := ParseDirName("***TS FILE*** Funny Girl (1968)")
	if len(e.Errata) == 0 || e.Errata[0].Kind != ErrataLeadingStar {
		t.Errorf("expected ErrataLeadingStar, got %v", e.Errata)
	}
	// Year still extracted correctly.
	if e.Year != 1968 {
		t.Errorf("expected year 1968, got %d", e.Year)
	}
}

func TestParseDirName_LeadingStarThreshold(t *testing.T) {
	// 1 or 2 asterisks should NOT trigger the flag.
	for _, name := range []string{"*Foo (2001)", "**Foo (2001)"} {
		e := ParseDirName(name)
		for _, f := range e.Errata {
			if f.Kind == ErrataLeadingStar {
				t.Errorf("ParseDirName(%q): unexpected ErrataLeadingStar", name)
			}
		}
	}
	// 3 or more should trigger it.
	for _, name := range []string{"***Foo (2001)", "****Foo (2001)"} {
		e := ParseDirName(name)
		found := false
		for _, f := range e.Errata {
			if f.Kind == ErrataLeadingStar {
				found = true
			}
		}
		if !found {
			t.Errorf("ParseDirName(%q): expected ErrataLeadingStar", name)
		}
	}
}

func TestParseDirName_InvalidYear(t *testing.T) {
	e := ParseDirName("Cartoons (Misc.)")
	if len(e.Errata) != 1 || e.Errata[0].Kind != ErrataInvalidYear {
		t.Errorf("expected ErrataInvalidYear, got %v", e.Errata)
	}
}

func TestParseDirName_LeadingThePlusMissingYear(t *testing.T) {
	// "The Foo" with no year: should get both a leading-The flag and a missing-year flag.
	e := ParseDirName("The Great Movie")
	kinds := make(map[ErrataKind]bool)
	for _, f := range e.Errata {
		kinds[f.Kind] = true
	}
	if !kinds[ErrataLeadingThe] {
		t.Error("expected ErrataLeadingThe flag")
	}
	if !kinds[ErrataMissingYear] {
		t.Error("expected ErrataMissingYear flag")
	}
}

// ─── Year extraction ──────────────────────────────────────────────────────────

func TestParseDirName_YearExtracted(t *testing.T) {
	cases := []struct {
		in       string
		wantYear uint16
	}{
		{"Wizard of Oz, The (1939)", 1939},
		{"TV Series (2005-2007)", 2005}, // range → first year
		{"Ongoing Show (2010...)", 2010},
		{"Cartoons Misc", 0},      // no year → 0
		{"Cartoons (Misc.)", 0},   // invalid year → 0
	}
	for _, tc := range cases {
		e := ParseDirName(tc.in)
		if e.Year != tc.wantYear {
			t.Errorf("ParseDirName(%q).Year = %d, want %d", tc.in, e.Year, tc.wantYear)
		}
	}
}

// ─── SortAlpha ────────────────────────────────────────────────────────────────

func TestSortAlpha(t *testing.T) {
	entries := []Entry{
		{DisplayTitle: "Zorro (1998)", SortKey: "zorro (1998)"},
		{DisplayTitle: "12 Angry Men (1957)", SortKey: "12 angry men (1957)"},
		{DisplayTitle: "Abbott (1940)", SortKey: "abbott (1940)"},
		{DisplayTitle: "100 Dalmatians (1961)", SortKey: "100 dalmatians (1961)"},
	}
	SortAlpha(entries)
	want := []string{"100 dalmatians (1961)", "12 angry men (1957)", "abbott (1940)", "zorro (1998)"}
	for i, e := range entries {
		if e.SortKey != want[i] {
			t.Errorf("position %d: got %q, want %q", i, e.SortKey, want[i])
		}
	}
}

// ─── SortByYear ───────────────────────────────────────────────────────────────

func TestSortByYear_Ascending(t *testing.T) {
	entries := []Entry{
		{SortKey: "zorro (1998)", Year: 1998},
		{SortKey: "12 angry men (1957)", Year: 1957},
		{SortKey: "wizard of oz, the (1939)", Year: 1939},
	}
	SortByYear(entries)
	wantYears := []uint16{1939, 1957, 1998}
	for i, e := range entries {
		if e.Year != wantYears[i] {
			t.Errorf("position %d: got year %d, want %d", i, e.Year, wantYears[i])
		}
	}
}

func TestSortByYear_NoYearAtEnd(t *testing.T) {
	entries := []Entry{
		{SortKey: "no year here", Year: 0},
		{SortKey: "abbott (1940)", Year: 1940},
		{SortKey: "also no year", Year: 0},
		{SortKey: "zorro (1998)", Year: 1998},
	}
	SortByYear(entries)
	// No-year entries must both be at the end.
	if entries[0].Year == 0 || entries[1].Year == 0 {
		t.Errorf("no-year entries should be at end, got years: %d %d %d %d",
			entries[0].Year, entries[1].Year, entries[2].Year, entries[3].Year)
	}
	if entries[2].Year != 0 || entries[3].Year != 0 {
		t.Errorf("expected last two entries to have Year==0, got %d and %d",
			entries[2].Year, entries[3].Year)
	}
}

func TestSortByYear_SameYearAlphaTiebreak(t *testing.T) {
	entries := []Entry{
		{SortKey: "zorro (1939)", Year: 1939},
		{SortKey: "abbott (1939)", Year: 1939},
	}
	SortByYear(entries)
	if entries[0].SortKey != "abbott (1939)" {
		t.Errorf("expected alpha tiebreak within same year, got %q first", entries[0].SortKey)
	}
}

func TestSortByYear_NoYearAlphaTiebreak(t *testing.T) {
	entries := []Entry{
		{SortKey: "zebra", Year: 0},
		{SortKey: "apple", Year: 0},
	}
	SortByYear(entries)
	if entries[0].SortKey != "apple" {
		t.Errorf("no-year entries should be alpha among themselves, got %q first", entries[0].SortKey)
	}
}
