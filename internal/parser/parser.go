// Package parser handles reading and interpreting movie directory names.
//
// Expected directory naming convention:
//
//	Title (YYYY)
//	Title (YYYY-YYYY)       — year range (e.g. TV series)
//	Title (YYYY...)         — open-ended range
//	Title, The (YYYY)       — cataloging-inversion form (preferred)
//
// Directories that deviate from these rules are still included in the output
// but are flagged for the errata section.
package parser

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// ErrataKind describes why an entry was flagged for the errata section.
type ErrataKind int

const (
	ErrataLeadingThe  ErrataKind = iota // "The Foo (1999)" → auto-corrected to "Foo, The (1999)"
	ErrataMissingYear                   // no year parenthetical found
	ErrataInvalidYear                   // parenthetical found but content is not a year
)

// ErrataFlag records a single issue with a directory entry.
type ErrataFlag struct {
	Kind    ErrataKind
	Message string // human-readable description for the errata section
}

// Entry represents one movie directory, parsed and ready for layout.
type Entry struct {
	// DisplayTitle is the title as it will appear in the movie list,
	// including the year (e.g. "Wizard of Oz, The (1939)").
	DisplayTitle string

	// SortKey is the lowercase DisplayTitle used for alphabetical ordering.
	SortKey string

	// Year is the first four-digit year extracted from the directory name.
	// 0 means no valid year was found (entry is flagged in errata).
	Year uint16

	// RawDir is the original directory name, unchanged.
	RawDir string

	// Errata holds any issues found during parsing. Empty means no issues.
	Errata []ErrataFlag
}

// ScanDirectory reads all subdirectory names from path and returns a parsed
// but unsorted slice of Entries. Non-directory entries are ignored.
// Entries with issues (missing year, leading "The", etc.) are still included
// and flagged. Call SortAlpha or SortByYear on the result before layout.
func ScanDirectory(path string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", path, err)
	}

	var entries []Entry
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		entries = append(entries, ParseDirName(de.Name()))
	}

	return entries, nil
}

// ParseDirName parses a single directory name into an Entry.
// It never returns an error; directories that don't match the expected format
// are returned as flagged entries.
func ParseDirName(name string) Entry {
	e := Entry{RawDir: name}

	// Step 1: detect and auto-correct leading "The".
	corrected, wasLeading := normalizeThe(name)
	if wasLeading {
		e.Errata = append(e.Errata, ErrataFlag{
			Kind:    ErrataLeadingThe,
			Message: fmt.Sprintf("directory starts with 'The' (auto-corrected): %q → %q", name, corrected),
		})
		name = corrected // work with the corrected form for the rest of parsing
	}

	// Step 2: find and validate the year parenthetical.
	yearInfo, yearErr := findYear(name)
	switch yearErr {
	case nil:
		// Happy path: extract title and year cleanly.
		title := strings.TrimRightFunc(name[:yearInfo.start], unicode.IsSpace)
		e.DisplayTitle = title + " " + name[yearInfo.start:yearInfo.end+1]
		// The first four characters inside '(' are guaranteed digits by isYearContent.
		inner := name[yearInfo.start+1 : yearInfo.end]
		var y uint16
		for _, b := range []byte(inner[:4]) {
			y = y*10 + uint16(b-'0')
		}
		e.Year = y

	case errMissingYear:
		// No parenthetical at all — include the raw name as the display title.
		e.DisplayTitle = strings.TrimRightFunc(name, unicode.IsSpace)
		e.Errata = append(e.Errata, ErrataFlag{
			Kind:    ErrataMissingYear,
			Message: fmt.Sprintf("no year found in directory name: %q", e.RawDir),
		})

	case errInvalidYear:
		// A parenthetical exists but doesn't look like a year.
		e.DisplayTitle = strings.TrimRightFunc(name, unicode.IsSpace)
		e.Errata = append(e.Errata, ErrataFlag{
			Kind:    ErrataInvalidYear,
			Message: fmt.Sprintf("parenthetical present but not a valid year: %q", e.RawDir),
		})
	}

	e.SortKey = strings.ToLower(e.DisplayTitle)
	return e
}

// ─── internal helpers ─────────────────────────────────────────────────────────

var (
	errMissingYear = fmt.Errorf("no year parenthetical found")
	errInvalidYear = fmt.Errorf("parenthetical is not a valid year")
)

// yearSpan holds the position of a year parenthetical within a string.
type yearSpan struct {
	start int // index of '('
	end   int // index of ')'
}

// findYear locates the last valid year parenthetical in s.
// It returns errMissingYear if none is found, errInvalidYear if a
// parenthetical is present but its content is not a year pattern.
//
// Accepted patterns inside the parentheses:
//
//	YYYY          e.g. (1999)
//	YYYY-YYYY     e.g. (2005-2007)
//	YYYY...       e.g. (2010...)
func findYear(s string) (yearSpan, error) {
	// Walk backwards so we pick the last '(' first — year is always at the end.
	foundInvalid := false
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != '(' {
			continue
		}
		close := strings.IndexByte(s[i:], ')')
		if close < 0 {
			continue
		}
		closeIdx := i + close
		inner := s[i+1 : closeIdx]

		if isYearContent(inner) {
			return yearSpan{start: i, end: closeIdx}, nil
		}
		// Found a parenthetical but it wasn't a year — note it and keep looking.
		foundInvalid = true
	}
	if foundInvalid {
		return yearSpan{}, errInvalidYear
	}
	return yearSpan{}, errMissingYear
}

// isYearContent returns true if s is a valid year / year-range / open-range.
func isYearContent(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return false
	}
	// Must start with 4 digits.
	if !allDigits(s[:4]) {
		return false
	}
	rest := s[4:]
	switch {
	case rest == "":
		return true // (YYYY)
	case rest == "...":
		return true // (YYYY...)
	case len(rest) == 5 && rest[0] == '-' && allDigits(rest[1:]):
		return true // (YYYY-YYYY)
	default:
		return false
	}
}

// allDigits returns true if s is non-empty and every byte is an ASCII digit.
func allDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, b := range []byte(s) {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// normalizeThe detects "The Title (YYYY)" (leading The) and converts it to
// "Title, The (YYYY)" (trailing The / cataloging-inversion form).
// Returns the (possibly unchanged) string and whether a conversion was made.
func normalizeThe(s string) (string, bool) {
	if !strings.HasPrefix(s, "The ") {
		return s, false
	}
	// Find the year parenthetical so we can split "The <rest> (YYYY)" correctly.
	// We want everything between "The " and the last ' (' as the base title.
	lastParen := strings.LastIndex(s, " (")
	if lastParen < 0 {
		// No year — just strip "The " and append ", The"
		rest := strings.TrimSpace(s[4:])
		return rest + ", The", true
	}
	base := strings.TrimSpace(s[4:lastParen])
	year := s[lastParen:] // " (YYYY)" including the leading space
	return base + ", The" + year, true
}

// SortAlpha sorts entries in-place using case-insensitive lexicographic order
// of SortKey. Numerals precede letters ('0'–'9' < 'a'–'z' in ASCII/Unicode),
// matching the traditional catalog sort seen in the old implementation.
func SortAlpha(entries []Entry) {
	// Insertion sort — correct and fast enough for slices < 10k entries.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].SortKey < entries[j-1].SortKey; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// SortByYear sorts entries in-place by ascending year. Entries with no valid
// year (Year == 0) are placed at the end. Ties — same year, or both no year —
// are broken alphabetically by SortKey.
func SortByYear(entries []Entry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && yearLess(entries[j], entries[j-1]); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// yearLess reports whether a should sort before b in year order.
func yearLess(a, b Entry) bool {
	switch {
	case a.Year == 0 && b.Year == 0:
		return a.SortKey < b.SortKey // both unknown → alpha
	case a.Year == 0:
		return false // a unknown → goes to end
	case b.Year == 0:
		return true // b unknown → a sorts first
	case a.Year != b.Year:
		return a.Year < b.Year
	default:
		return a.SortKey < b.SortKey // same year → alpha tiebreak
	}
}
