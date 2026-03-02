# movDB Changelog

This file documents the development history of movDB as a reference for future
work. It is intentionally detailed to aid agentic continuation of this project.

---

## Origin

movDB is a Go rewrite of an older Zig project previously available at (`~/Projects/zig/movieDb`)
but it is subject to deprecation and the folder may disappear at any time.
The Zig version produced tab-separated two-column plain-text output and
hard-truncated titles at 31 characters (mid-word). The rewrite fixed this and
added many new features.

---

## v0.1 — Initial Go implementation

**Architecture** (`internal/` package layout):

- `internal/parser` — reads subdirectory names, parses titles and years
- `internal/layout` — word-wraps titles, paginates into a two-column grid
- `internal/render` — generates output (HTML and Typst backends)
- `cmd/movdb` — CLI entry point

**Parser (`internal/parser/parser.go`)**

- `Entry` struct: `DisplayTitle`, `SortKey`, `Year uint16`, `RawDir`, `Errata []ErrataFlag`
- `ErrataKind` iota: `ErrataLeadingStar`, `ErrataLeadingThe`, `ErrataMissingYear`, `ErrataInvalidYear`
- `ScanDirectory(path)` — reads subdirs, returns unsorted `[]Entry`
- `ParseDirName(name)` — never errors; flags issues via `Errata`
- Auto-corrects leading "The": `"The Foo (1999)"` → `"Foo, The (1999)"` (flagged in errata)
- Year extraction: last `(YYYY)`, `(YYYY-YYYY)`, or `(YYYY...)` parenthetical
- `SortAlpha` — case-insensitive insertion sort on `SortKey`
- `SortByYear` — ascending year, `Year==0` entries at end, alpha tiebreak

**Layout (`internal/layout/layout.go`)**

- Constants: `NumberWidth=5`, `SepWidth=2`, `TitleWidth=36`, `RowsPerCol=54`, `ColWidth=43`
- `WrappedEntry` struct: `Number`, `Lines []string`, `IsIssue`, `RawDir`, `Errata`, `Year uint16`
  - `Year` added later for year-range page headers
- `BuildPages(entries)` — wraps titles, enforces entry integrity (no entry splits across columns/pages)
- `WrapTitle(title, width)` — greedy word-wrap, never mid-word breaks
- `FormatEntryLines(e)` — fixed-width formatted strings for HTML renderer

**HTML renderer (`internal/render/render.go`)**

- `Config` struct: `Title string`, `Date string`, `ByYear bool`
- `RenderHTML(pages, errataEntries, cfg)` — full self-contained HTML document
- US Letter via CSS `@page { size: letter portrait; margin: 0.75in; }`
- Courier New 9pt, `43ch` column width, `line-height: 12pt`
- Zebra striping: odd-numbered entries get `background-color: #e8e8e8`
- Errata section only rendered when `len(errataEntries) > 0`

**Typst renderer (`internal/render/typst.go`)**

- `RenderTypst(entries, cfg)` — does NOT use `layout.BuildPages`; Typst handles pagination
- Font: Linux Libertine O 10pt (`pkgs.libertine` in nix devShell)
- `TYPST_FONT_PATHS` must point to libertine's opentype dir (set in `flake.nix` shellHook)
- `#block(breakable: false)` ensures entries never split across columns/pages
- `#columns(2, gutter: 0.4in)` for two-column layout
- Zebra striping: odd entries get `fill: luma(232)`, `inset: (x: 2pt, y: 3pt)` on all entries
- `#set par(leading: 4pt, spacing: 0pt)` — spacing is inside block inset, not between blocks
- Errata section only rendered when `len(errata) > 0`

**CLI (`cmd/movdb/main.go`)**

- Flags: `-title`, `-date`, `-o`, `-y` (sort by year), `-fmt html|typst`
- Flags must come BEFORE the positional directory argument (Go `flag` package stops at first non-flag)
- Typst stderr message includes compile hint: `typst compile <file>`

**Nix flake (`flake.nix`)**

- `buildGoModule` with `vendorHash = null` (no external Go dependencies)
- `nix run github:ilioscio/movDB -- [flags] <dir>` — runs movdb directly
- `nix run github:ilioscio/movDB#pdf -- -o out.pdf <dir>` — generates PDF via `movdb-pdf` wrapper
- `movdb-pdf`: `writeShellScriptBin` that pipes `movdb -fmt typst` into `typst compile`
  - Intercepts `-o` flag; default output filename is `list.pdf`
  - Pins both `movdb` and `typst`/`libertine` — no host tooling required
- `overlays.default` and `nixosModules.default` with `programs.movdb.enable` option
- `devShells.default`: `go`, `gopls`, `gotools`, `typst`, `libertine`
- `flake.lock` must be committed for `nix run github:...` to work

---

## Feature: Page range in header

Each page header shows the alphabetical (or year) range of entries on that
page, e.g. `'Dark' to 'Dr'`.

**Implementation**

- `titlePrefix(s string) string` in `render.go`:
  - Takes first whitespace-separated word of the title
  - Strips trailing `,` and `.` (e.g. `"Mask,"` → `"Mask"`, `"Dr."` → `"Dr"`)
  - Does NOT strip `!` or `?`
- `pageRangeParts(p, byYear) (first, last string)` in `render.go`:
  - When `byYear=false`: uses `titlePrefix` on first/last entry `Lines[0]`
  - When `byYear=true`: uses `yearLabel(entry.Year)` — `"????"` for year 0
- HTML: range rendered as `'first' to 'last'` with literal `'` chars outside
  `html.EscapeString` (Go's `html.EscapeString` encodes `'` as `&#39;`)
- Typst: `#metadata("word")<entry-mark>` emitted before each entry; header
  uses `query(<entry-mark>).filter(m => m.location().page() == here().page())`
  to find first/last on each page; range assembled in Typst as a string
- `typstStringEscape(s)` escapes `\` and `"` for Typst string literals
- `ByYear bool` field added to `render.Config`; wired from `-y` flag in main

---

## Feature: Header layout

**Layout**: 3-column CSS grid / Typst grid — `1fr auto 1fr`

- Left `1fr`: title + date (flex row, `gap: 1.5em`)
- Center `auto`: range (bold, truly centered because side columns are equal)
- Right `1fr`: page number (right-aligned)

**Mirrored running heads for duplex printing**

- Odd pages (recto): `[title · date] | [range] | [Page N]` — page# at outer right
- Even pages (verso): `[Page N] | [range] | [title · date]` — page# at outer left
- HTML: `p.Number % 2 == 0` branch in `writePage`; even pages use `.header-right`
  class (`justify-content: flex-end`) for the title+date group
- Typst: `#if calc.even(here().page()) [...] else [...]` in header context
- Format args use Go's `%[1]s`/`%[2]s` indexed verbs to reuse title/date
  across both grid branches without repeating arguments

**Header font size**: 12pt (body is 10pt) — set via `#set text(size: 12pt)` in
header context; HTML header uses `font-size: 11pt` on `.page-header`

---

## Feature: Errata page label

Errata pages show **"Errata"** bold in the center (range position) instead of
a range label.

- Typst: `#let in-errata = state("in-errata", false)` declared before `#set page`
  - `#in-errata.update(true)` emitted by `writeTypstErrata` before `#pagebreak()`
  - Header checks `in-errata.at(here())` first; if true, `range-str = "Errata"`
  - Handles multi-page errata sections correctly (state persists)
- HTML: `writeErrata` emits `<span class="header-range">Errata</span>` directly
- Errata page title no longer includes `"— Errata"` suffix (the centered label does that job)

---

## Feature: Zebra striping

Alternating shaded backgrounds on odd-numbered entries (1, 3, 5, …).

- HTML: `class="entry shaded"` on odd entries; CSS `.entry.shaded { background-color: #e8e8e8; }`
- Typst: `fill: if shaded { luma(232) } else { none }` in entry block
  - All entries have `inset: (x: 2pt, y: 3pt)` so fill covers padding area
  - `#set par(spacing: 0pt)` moves inter-entry gap inside the fill region

---

## Feature: ErrataLeadingStar

Directories beginning with 3 or more `*` characters are flagged for processing.
These are intentionally placed at the list beginning (sort before letters) and
need manual attention before being properly catalogued.

- Detection in `ParseDirName` as step 1 (before The-normalization)
- `ErrataLeadingStar` is the first `iota` constant so its errata section renders first
- Threshold: exactly 3+ asterisks; 1 or 2 are ignored
- Errata section heading: "Directories that require processing"
- Entries still appear in the main list at the top (sort order preserved)

---

## Known gotchas / sharp edges

- `vendorHash = null` in `buildGoModule` means **no external dependencies**,
  NOT "use the vendor directory". Adding deps requires computing a real hash.
- `flake.lock` must be committed for `nix run github:...` to work remotely.
- Typst does not auto-discover fonts from the Nix store; `TYPST_FONT_PATHS`
  must be set (done in `flake.nix` `shellHook`).
- Linux Libertine font family name in Typst is `"Linux Libertine O"` (with O).
- Go's `flag` package stops parsing at the first non-flag argument. Flags
  must precede the directory path: `movdb -fmt typst /path` not `movdb /path -fmt typst`.
- `html.EscapeString` encodes single quotes as `&#39;`. Literal `'` chars
  used in range labels must be placed in the format string, not the escaped string.
- The errata page is suppressed entirely when there are no flagged entries
  (both HTML and Typst gate on `len(errata) > 0`).
- The `movdb-pdf` wrapper parses `-o` itself in shell; all other flags pass
  through to `movdb` unchanged.
