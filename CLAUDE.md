# CLAUDE.md

Guidance for working in this repository.

## What this is

`yaml-fixed` is a YAML parser/emitter (Go) whose defining rule is: **indentation
is tabs, and only tabs.** A space in the indentation region of a line is a
syntax error. Spaces remain legal inside values, after `key:`/`-` separators,
in quotes, and in flow collections. If you change parsing behaviour, keep that
rule absolute.

## Layout

- `tabyaml/` -- the library package (`package tabyaml`).
  - `parse.go` -- line scanning, tab-indent enforcement (`measure`), and the
    recursive block parser (`Parse`, `ParseAll`).
  - `scalar.go` -- scalar typing (1.2 core schema), quoting/unquoting, comments.
  - `flow.go` -- flow collections `[...]` / `{...}`.
  - `block.go` -- literal `|` and folded `>` block scalars with chomping.
  - `encode.go` -- `Marshal` (reflection -> tab-indented output).
  - `decode.go` -- `Unmarshal` (reflection into structs/maps/slices) and shared
    reflection helpers.
  - `errors.go` -- `SyntaxError` (line/col) and `TypeError`.
- `cmd/tabyaml/` -- the cobra CLI. One command per file, each self-registering
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

- Children are indented strictly more tabs than their parent; nested blocks must
  be deeper than the `key:`/`-` they hang off (no same-indent sequences).
- Multi-key mappings inside sequences use the expanded form (dash alone, body
  one tab deeper); `Marshal` emits this form so output re-parses.
- Anchors/aliases, merge keys, and explicit tags are intentionally unsupported.
