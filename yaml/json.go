package yaml

import "strings"

// This file holds the JSON-consumption path. YAML is a superset of JSON, and a
// JSON document is just a flow collection (or scalar): its structure is carried
// by the delimiters {}, [], "," and ":", never by indentation. That makes JSON
// parseable regardless of how its lines are laid out, which is why this library
// -- otherwise strict that indentation must be tabs -- accepts space-indented
// JSON, with a one-per-file Warn. ParseAll routes such documents here.

// documentIsFlow reports whether a document's first structural line begins a
// flow collection ("{" or "["), making the whole document JSON-style. Leading
// blank and whole-line comment lines are skipped; indentation is ignored so
// that the detection still succeeds for space-indented JSON (which measure
// would otherwise reject before we ever get here).
func documentIsFlow(lines []physLine) bool {
	for _, ln := range lines {
		t := strings.TrimLeft(ln.text, " \t")
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return t[0] == '{' || t[0] == '['
	}
	return false
}

// parseFlowDocument consumes an entire JSON-style document by joining its
// physical lines and parsing them as a single flow value. Because flow
// structure is delimiter-based, the join needs no indentation handling at all:
// the flow parser treats line breaks as ordinary whitespace. It reports whether
// any line used space indentation so the caller can warn once per file.
func parseFlowDocument(lines []physLine) (value any, spacedIndent bool, err error) {
	var b strings.Builder
	first := 0
	started := false
	for _, ln := range lines {
		if !started {
			// Skip blank and whole-line comment lines that precede the
			// collection; parseFlow only tolerates a comment after the value.
			t := strings.TrimLeft(ln.text, " \t")
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			started = true
			first = ln.no
		} else {
			b.WriteByte('\n')
		}
		b.WriteString(ln.text)
		if hasSpaceIndent(ln.text) {
			spacedIndent = true
		}
	}
	v, err := parseFlow(b.String(), first)
	if err != nil {
		return nil, spacedIndent, err
	}
	return v, spacedIndent, nil
}

// hasSpaceIndent reports whether a line's leading indentation contains a space.
// Tabs (the canonical indentation) do not count; only a space in the indent
// region -- the hallmark of pretty-printed JSON -- does. Blank lines and lines
// with no indentation return false.
func hasSpaceIndent(line string) bool {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			return true
		case '\t':
			// still scanning the indentation region
		default:
			return false
		}
	}
	return false
}
