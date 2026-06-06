# CLAUDE.md

Guidance for working in this repository.

## What this is

`yaml-fixed` is a YAML parser/emitter (Go). It is an ordinary YAML library
except in one respect: indentation is done with tabs, not spaces. A line's
structural depth is its count of leading tabs and nothing else; spaces after the
tabs are alignment and never change depth. Leading spaces with no preceding tab
are a syntax error, as is a tab after alignment spaces. Spaces are otherwise
legal inside values, after `key:`/`-` separators, in quotes, and in flow
collections. If you change parsing behaviour, keep that rule absolute (`measure`
in `parse.go` enforces it).

## Layout

- `yaml/` -- the library package (`package yaml`).
  - `parse.go` -- line scanning, tab-indent enforcement (`measure`), and the
    recursive block parser (`Parse`, `ParseAll`).
  - `scalar.go` -- scalar typing (1.2 core schema), quoting/unquoting, comments.
  - `flow.go` -- flow collections `[...]` / `{...}`.
  - `block.go` -- literal `|` and folded `>` block scalars with chomping.
  - `encode.go` -- `Marshal` (reflection -> tab-indented output).
  - `decode.go` -- `Unmarshal` (reflection into structs/maps/slices) and shared
    reflection helpers.
  - `errors.go` -- `SyntaxError` (line/col) and `TypeError`.
- `cmd/yaml/` -- the cobra CLI. One command per file, each self-registering
  via `init()`; `main.go` only calls `Execute()`.

## Build / test

Always use the `go-toolchain` wrapper (never bare `go build`/`go test`):

```console
$ go-toolchain
```

It runs `go mod tidy`, `go vet`, tests with coverage, and the build. CI
(`.github/workflows/ci.yml`) runs the same via `wow-look-at-my/go-toolchain@v1`.

- Tests use `testify` (`require`/`assert`). A repo lint hook rewrites tests into
  that style and gofmt's with tabs -- write new tests in testify style to avoid
  churn.
- Coverage is gated at 80%. Keep new code covered.

## Design decisions worth preserving

- Children have strictly more tabs than their parent; depth is tab count only,
  so alignment spaces never create nesting (this is what lets a sequence item's
  mapping body align past the `- ` marker while staying at the item's depth).
- A sequence item's body is gathered by `collectItemBody` (deeper lines, plus
  same-depth non-dash lines) and parsed via a sub-parser in `parseItemBody`.
- `Marshal` emits the compact aligned form for mappings in sequences at depth
  >= 1, and the dash-on-its-own-line form at the left margin (depth 0, where
  there is no tab to align against); both re-parse.
- Anchors/aliases, merge keys, and explicit tags are intentionally unsupported.
