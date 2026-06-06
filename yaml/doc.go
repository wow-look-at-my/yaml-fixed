// Package yaml implements a YAML parser and emitter for Go.
//
// It is an ordinary YAML library with one deliberate difference from most
// others: indentation is done with tabs, not spaces. A line's structural depth
// is its number of leading TAB characters, and nothing else. After those tabs
// you may add spaces to align content -- for instance to line a mapping up past
// a "- " marker -- and those spaces never change the depth. Using spaces as
// indentation is the error it ought to be: leading spaces with no preceding tab
// are a syntax error, as is a tab placed after alignment spaces (indent first,
// then align). Spaces are otherwise perfectly legal -- inside scalar values,
// after the "key:" separator, after a "-" sequence marker, inside quotes, and
// inside flow collections.
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
// A multi-key mapping inside a sequence is written with one tab of indentation
// for the item and spaces to align the keys past the "- " marker:
//
//	people:
//		- name: Alice
//		  age: 30
//		- name: Bob
//		  age: 25
//
// Here every line under people carries one leading tab (the structural depth);
// the two spaces before "age" are alignment, not depth, so "name" and "age" are
// siblings of the same mapping. To make a value a child instead of a sibling,
// give it another tab. A scalar (- value), a flow (- [1, 2]) and a dash on its
// own line (with the body on following lines) are all accepted too.
//
// # Not supported
//
// Anchors/aliases (&, *), the merge key (<<), explicit tags (!!str) and
// directives other than being skipped (%YAML, %TAG) are intentionally omitted.
package yaml
