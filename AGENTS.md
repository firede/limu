# Working on limu

limu reads Mermaid diagrams and lays them out in Go; backends draw what it lays out. Read
docs/design.md first: it says what each stage owns and why style is kept out of layout.

## Rules

- The core packages (`text`, `scene`, `style`, `layout`, `svg` and the diagrams) import the
  standard library only. A backend that needs another module, as one for MyGo does, is a
  module of its own, with its own go.mod.
- Layout never decides a color, and style never moves a thing. A setting that changes the
  size of anything belongs to `layout.Options`.
- `./scripts/verify.sh` passes before a change is done.

## Porting from Pele

- Port function for function, keeping Pele's names where Go allows, so the two can be read
  side by side. A Go function ported from Pele says which in its comment ("as Pele's
  layoutLabel does").
- Keep JavaScript's meaning where it reaches what is measured or placed: strings counted
  in UTF-16 code units, `\s` and `trim` with Unicode's spaces (`text/js.go`), `Math.round`
  rounding halves up, the order floating-point sums are taken in, V8's `Math.hypot`, stable
  sorts, and Map and Set iterating in the order keys were added. Regular expressions are
  written out by hand, with the backtracking they would do.
- Go may fuse `x*y + z` into one multiply-add with a single rounding, and does on arm64;
  JavaScript never does. Where a product is added or subtracted and the result must match
  Pele to the bit, write the product as `float64(x*y)`, which Go rounds on its own.
- Every ported stage is checked against Pele. A suite in `internal/oracle/suites` (a module
  whose default export is given helpers to import Pele's sources, read cases from
  `internal/oracle/cases` and write fixtures) writes what Pele makes of the cases;
  `./scripts/oracle.sh <pele> [suite…]` runs suites, and the package's tests compare with
  the fixture in its `testdata/pele`. Compare numbers exactly; a difference in the last bit
  is a difference in the port.
- The port follows Pele 1.6.0, commit 44da152. To follow a newer Pele, write the fixtures
  anew from it and port what its diff changed.

## Writing

Comments and documents are plain and exact: say what a thing does and why, in full
sentences, without selling it. Commit messages say what changed, in the imperative.
