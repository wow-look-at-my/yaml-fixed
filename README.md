# yaml-fixed

A YAML parser and emitter for Go.

It is an ordinary YAML library, with one difference: it does not have the bug
where spaces are used for indentation. Indentation is done with tabs -- a line's
depth is its number of leading tabs, and nothing else. Spaces after the tabs are
alignment and never change the depth, and spaces with no preceding tab in the
indentation region are the syntax error they ought to be.

```
# This is valid YAML (every indent below is a single TAB):
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
# => yaml: line 2, column 1: spaces cannot be used for indentation; indent with tabs (spaces only align after a tab)
```

Spaces are still legal everywhere they actually belong: inside scalar values
(`name: Jane Doe`), after the `key:` separator, after a `-` marker, inside
quotes, inside flow collections (`[1, 2, 3]`), and as alignment after the
leading tabs. A child node has strictly more tabs than its parent; that is the
whole indentation model.

### Consuming JSON

There is one principled exception: **JSON**. YAML is a superset of JSON, and a
JSON document is just a flow collection -- its structure comes from its
delimiters (`{}`, `[]`, `,`, `:`), not from indentation, so it parses no matter
what whitespace is present. A document whose top-level value starts with `{` or
`[` is therefore consumed as one flow value, even when it is the usual
space-indented, multi-line, pretty-printed JSON:

```console
$ printf '{\n  "name": "Ada",\n  "port": 8080\n}\n' | yaml to-json
yaml: warning: accepted spaces for indentation while consuming JSON; JSON structure comes from its delimiters, not whitespace, so the indentation was ignored (indent with tabs to silence this)
{
  "name": "Ada",
  "port": 8080
}
```

Because the spaces genuinely cannot change a JSON document's meaning, they are
accepted -- with a single warning per file. Indent the JSON with tabs and the
warning goes away. This applies *only* to JSON-style (flow) documents; ordinary
block YAML still rejects space indentation as the error it ought to be.

There is exactly one parser: `from-json` reads JSON with `yaml.Parse`, the same
parser `to-json`, `validate`, and `fmt` use, rather than a separate JSON
decoder. JSON in, JSON out, JSON converted -- all the same code path.

Library users can redirect or silence the warning by replacing the package-level
`yaml.Warn` hook (it defaults to writing one line to standard error):

```go
yaml.Warn = func(msg string) {} // silence
```

## Library

```go
import "github.com/wow-look-at-my/yaml-fixed/yaml"
```

### Parse into a generic value

```go
v, err := yaml.Parse([]byte("a: 1\nb:\n\t- x\n\t- y\n"))
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
err := yaml.Unmarshal(src, &cfg)
```

Fields are matched by the `yaml:"..."` tag, falling back to the lower-cased
field name. `,omitempty` and `-` are honoured by the encoder.

### Marshal

```go
out, err := yaml.Marshal(map[string]any{
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
go install github.com/wow-look-at-my/yaml-fixed/cmd/yaml@latest
```

| Command | Description |
|---|---|
| `yaml validate [file]` | Exit 0 if the input is well-formed YAML, else report the line/column. |
| `yaml fmt [file] [-w]` | Canonicalise: sort keys, re-indent with tabs. `-w` rewrites the file. |
| `yaml to-json [file]` | Convert YAML to JSON. |
| `yaml from-json [file]` | Convert JSON to YAML. |

Every command reads the named file, or standard input when given no file (or `-`).

```console
$ printf 'server:\n\thost: localhost\n' | yaml to-json
{
  "server": {
    "host": "localhost"
  }
}
```

## Sequences of mappings

The item is indented with a tab; the keys are aligned past the `- ` marker with
spaces. `name` and `age` carry the same single tab, so they are siblings:

```
people:
	- name: Alice
	  age: 30
	- name: Bob
	  age: 25
```

The dash may also stand on its own line with the body below it; either way the
body sits at the item's tab depth, aligned with spaces:

```
people:
	-
	  name: Alice
	  age: 30
```

Because depth is counted in *tabs only*, alignment spaces never create nesting.
To make a value a child rather than a sibling, give it another **tab**:

```
people:
	- name:
			first: Alice
			last: Liddell
	  age: 30
```

`- scalar` and `- [1, 2]` work too. `Marshal` emits the compact aligned form
(and, at the document's left margin where there is no tab to align against, the
dash-on-its-own-line form) so output always re-parses.

## Supported syntax

- Block mappings and block sequences, nested with tabs.
- Typed plain scalars, single- and double-quoted scalars (with escapes).
- Flow collections: `[a, b]` and `{a: 1, b: 2}`, on one line or spanning several
  (so whole JSON documents parse; see [Consuming JSON](#consuming-json)).
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

- Library: `yaml/`
- CLI: `cmd/yaml/`
