# yaml-fixed

A YAML parser and emitter that uses tabs, **and only tabs**, for indentation.

The YAML spec indents with spaces. That is a mistake. Indentation expresses one
thing -- depth -- and a tab is the single character whose entire purpose is
"advance one indentation level." Its width is a property of *your* editor, not
of the file, so everyone reads the same document at whatever indent width they
like without changing a byte. A space is a unit of horizontal text; holding it
down N times to imitate one level of depth is an encoding accident. `yaml-fixed`
treats that accident as a syntax error.

```
# This is valid tab-YAML (every indent below is a single TAB):
server:
	host: localhost
	port: 8080
	tls:
		enabled: true
```

```
# This is rejected -- the indentation uses spaces:
server:
  host: localhost
# => tabyaml: line 2, column 1: spaces cannot be used for indentation; tab-YAML indents with tabs only
```

Spaces are still perfectly legal everywhere they actually belong: inside scalar
values (`name: Jane Doe`), after the `key:` separator, after a `-` marker,
inside quotes, and inside flow collections (`[1, 2, 3]`).

## The one rule

> Structural indentation is one or more leading **tab** characters. A space in
> the indentation region of a line is an error, not a smaller indent.

A child node is indented with strictly more tabs than its parent. That is the
whole model.

## Library

```go
import "github.com/wow-look-at-my/yaml-fixed/tabyaml"
```

### Parse into a generic value

```go
v, err := tabyaml.Parse([]byte("a: 1\nb:\n\t- x\n\t- y\n"))
// v == map[string]any{"a": 1, "b": []any{"x", "y"}}
```

Scalars resolve with the YAML 1.2 core schema: `null`/`~`, `true`/`false`,
integers (decimal, `0x`, `0o`), floats (including `.inf`/`.nan`), everything else
a string.

### Unmarshal into a struct

```go
type Config struct {
	Name    string   `yaml:"name"`
	Port    int      `yaml:"port"`
	Modules []string `yaml:"modules"`
}

var cfg Config
err := tabyaml.Unmarshal(src, &cfg)
```

Fields are matched by the `yaml:"..."` tag, falling back to the lower-cased
field name. `,omitempty` and `-` are honoured by the encoder.

### Marshal (always tab-indented)

```go
out, err := tabyaml.Marshal(map[string]any{
	"server": map[string]any{"host": "localhost", "port": 8080},
})
// server:
// \thost: localhost
// \tport: 8080
```

Strings that would otherwise be read back as a number, boolean, or null are
quoted automatically, so `Marshal` then `Parse` round-trips.

## CLI

```
go install github.com/wow-look-at-my/yaml-fixed/cmd/tabyaml@latest
```

| Command | Description |
|---|---|
| `tabyaml validate [file]` | Exit 0 if the input is well-formed tab-YAML, else report the line/column. |
| `tabyaml fmt [file] [-w]` | Canonicalise: sort keys, re-indent with tabs. `-w` rewrites the file. |
| `tabyaml to-json [file]` | Convert tab-YAML to JSON. |
| `tabyaml from-json [file]` | Convert JSON to tab-YAML. |

Every command reads the named file, or standard input when given no file (or `-`).

```console
$ printf 'server:\n\thost: localhost\n' | tabyaml to-json
{
  "server": {
    "host": "localhost"
  }
}
```

## Sequences of mappings

A tab cannot align to the column just after `- `, so a multi-key mapping inside a
sequence is written in **expanded form**: the dash sits on its own line and the
mapping body is indented one further tab.

```
people:
	-
		name: Alice
		age: 30
	-
		name: Bob
		age: 25
```

Single inline values after a dash are fine: `- scalar`, `- key: value`,
`- [1, 2]`. `Marshal` always emits the expanded form so output re-parses.

## Supported syntax

- Block mappings and block sequences, nested with tabs.
- Typed plain scalars, single- and double-quoted scalars (with escapes).
- Flow collections: `[a, b]` and `{a: 1, b: 2}`.
- Block scalars: literal `|` and folded `>`, with `-`/`+` chomping.
- `#` comments (whole-line and trailing).
- Multiple documents separated by `---`, terminated by `...`.

### Not supported (intentionally)

Anchors/aliases (`&`, `*`), the merge key (`<<`), explicit tags (`!!str`), and
YAML directives beyond being skipped.

## Development

This is a Go module. Use the `go-toolchain` wrapper for everything (it builds,
tests, and reports coverage):

```console
$ go-toolchain
```

- Library: `tabyaml/`
- CLI: `cmd/tabyaml/`
