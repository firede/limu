# Working on limu

limu reads Mermaid diagrams and lays them out in Go, and backends draw what it lays out.
Read docs/design.md first. It says what each stage owns and why style is kept out of
layout.

## Rules

- The core packages import the standard library only. They are `text`, `scene`, `style`,
  `layout`, `svg`, the diagrams, and what the diagrams share under `diagram/internal`
  (`graphs`, `draw`, `css` and `common`). A backend that needs another module, as the one
  for MyGo does, is a module of its own with its own go.mod.
- Layout never decides a color, and style never moves anything. A setting that changes the
  size of anything belongs in `layout.Options`.
- `./scripts/verify.sh` passes before a change is done.

## Porting from Pele

- Port function for function, keeping Pele's names where Go allows, so the two can be read
  side by side. A Go function ported from Pele names its source in its comment ("as Pele's
  layoutLabel does").
- Keep JavaScript's meaning wherever it affects what is measured or placed. That means
  strings counted in UTF-16 code units, `\s` and `trim` with Unicode's spaces
  (`internal/chars`), `Math.round` rounding halves up, the order floating-point sums are
  taken in, V8's `Math.hypot`, stable sorts, and Map and Set iterating in insertion order.
  Write regular expressions out by hand, with the backtracking they would do.
- Go may fuse `x*y + z` into one multiply-add with a single rounding, and does on arm64 and
  on amd64 at `GOAMD64=v3`. JavaScript never does. Where a product is added or subtracted
  and the result must match Pele to the bit, write the product as `float64(x*y)`, which Go
  rounds on its own. Go also fuses across a call it inlines. A small function that adds to
  or subtracts from a parameter converts the parameter there, as `jsmath.Round` does, so a
  product passed in stays apart.
- Every ported stage is checked against Pele.
  - A suite in `internal/oracle/suites` writes what Pele makes of its cases. It's a module
    whose default export receives helpers to import Pele's sources and write fixtures. It
    holds or builds its cases, or for the first suites reads them from
    `internal/oracle/cases`.
  - What suites share is in `internal/oracle/lib`. What the oracle patches into Pele's
    sources is in the table in `internal/oracle/hooks.mjs`.
  - A suite `x` writes `x.json` in the package `x`, and `x-y` writes `y.json` there.
    `run.mjs` lists the exceptions, such as `graphs-layout` and `graphs-lanes`.
  - `./scripts/oracle.sh <pele> [suite…]` runs suites, on arm64 only. A package's tests
    compare with the fixture in its `testdata/pele`, read with `fixture.Load`.
  - Compare numbers exactly. A difference in the last bit is a difference in the port.
- A `render.json` holds the lines of its first `fixture.FullCases` cases and a digest of
  the lines of each case after them. Tests compare with `fixture.LoadDrawings` and
  `Drawings.Diff`. A test that reads Pele's lines through limu's code before comparing
  needs its suite to keep in full (`keep`) every line that reading changes, and the reading
  must change no other line. To see which line differs from a digest, run
  `./scripts/oracle.sh --full <pele> <suite>`, which writes the full lines to a file git
  ignores.
- Matching Pele to the last bit is how the port is checked, not what limu is for. Go as far
  as porting the function JavaScript calls, as `internal/jsmath` ports V8's sine. Don't
  write code whose only use is to match a platform's library to the last bit, as copying
  the C library's `pow` would be. Where such a difference can't be avoided, note it as a
  difference of the platform and leave the cases it affects out of the comparison.
- The port follows Pele 1.8.0, commit 8fad23b. To follow a newer Pele, write the fixtures
  anew from it and port what its diff changed.

## Writing

Comments and documents are plain and exact. Say what a thing does and why, in full
sentences, without selling it. In documents, write short, natural sentences, and avoid
semicolons and lists led by a colon in running prose. Commit messages say what changed, in
the imperative.
