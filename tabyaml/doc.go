// Package tabyaml implements a YAML parser and emitter that uses tabs, and
// only tabs, for indentation.
//
// # Why tabs
//
// Indentation expresses one thing: depth. A tab is the one character whose
// entire job is "advance to the next indentation stop." Its on-screen width is
// a property of the reader's environment, not of the file, so every developer
// can view the same document at the indent width they prefer without touching a
// byte. A space is a unit of horizontal text; pressing it N times to fake an
// indentation level is an encoding accident, not a design. tabyaml treats that
// accident as an error.
//
// The rule is therefore simple and absolute: structural indentation is one or
// more leading TAB characters. A space appearing in the indentation region of a
// line is a syntax error, not a smaller indent. Spaces remain perfectly legal
// everywhere they belong -- inside scalar values, after the "key:" separator,
// after a "-" sequence marker, inside quotes, and inside flow collections.
//
// # Supported syntax
//
//   - Block mappings:   key: value
//   - Block sequences:  - item
//   - Arbitrary nesting via additional leading tabs (a child has strictly more
//     tabs than its parent).
//   - Typed plain scalars: null/~, true/false, integers (decimal, 0x, 0o),
//     floats (incl. .inf/.nan), and strings.
//   - Single- and double-quoted scalars (with the usual escape sequences).
//   - Flow collections: [a, b, c] and {a: 1, b: 2}.
//   - Block scalars: literal "|" and folded ">" (with -, + chomping).
//   - Comments introduced by "#".
//   - Multiple documents separated by "---" (and terminated by "...").
//
// # Sequences of mappings
//
// Because a tab cannot align to the column after "- ", a multi-key mapping
// inside a sequence is written in expanded form: the dash stands alone and the
// mapping body is indented one further tab.
//
//	people:
//		-
//			name: Alice
//			age: 30
//		-
//			name: Bob
//			age: 25
//
// A single inline pair (- key: value), a scalar (- value) and a flow
// (- [1, 2]) are all accepted directly after the dash.
//
// # Not supported
//
// Anchors/aliases (&, *), the merge key (<<), explicit tags (!!str) and
// directives other than being skipped (%YAML, %TAG) are intentionally omitted.
package tabyaml
