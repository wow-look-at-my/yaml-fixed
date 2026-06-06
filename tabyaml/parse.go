package tabyaml

import (
	"strings"
)

// Parse decodes a single tab-YAML document into a generic Go value
// (map[string]any, []any, string, int, float64, bool, or nil).
//
// An empty input (or one consisting only of blank lines and comments) decodes
// to nil. If the input holds more than one document, Parse returns an error;
// use ParseAll for multi-document streams.
func Parse(data []byte) (any, error) {
	docs, err := ParseAll(data)
	if err != nil {
		return nil, err
	}
	switch len(docs) {
	case 0:
		return nil, nil
	case 1:
		return docs[0], nil
	default:
		return nil, errorf(0, 0, "input contains %d documents; use ParseAll", len(docs))
	}
}

// ParseAll decodes every document in a tab-YAML stream. Documents are separated
// by a "---" line and may be terminated by a "...". Documents that contain no
// content are skipped.
func ParseAll(data []byte) ([]any, error) {
	all := splitLines(data)
	docs := splitDocuments(all)

	var out []any
	for _, doc := range docs {
		p := &parser{lines: doc}
		if !p.hasContent() {
			continue
		}
		v, err := p.parseDocument()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// physLine is one physical input line together with its original 1-based number,
// so that errors can point at the source even after documents are split apart.
type physLine struct {
	text string
	no   int
}

// splitLines breaks the input into physical lines, normalising CRLF and CR to
// LF and dropping a trailing newline so it does not produce a spurious empty
// final line.
func splitLines(data []byte) []physLine {
	s := string(data)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if s == "" {
		return nil
	}
	raw := strings.Split(s, "\n")
	// A trailing newline yields a final empty element; drop it.
	if n := len(raw); n > 0 && raw[n-1] == "" {
		raw = raw[:n-1]
	}
	lines := make([]physLine, len(raw))
	for i, t := range raw {
		lines[i] = physLine{text: t, no: i + 1}
	}
	return lines
}

// splitDocuments partitions physical lines into documents at "---" markers and
// "..." terminators. A "--- value" marker carries inline content into the new
// document. Directive lines ("%...") are dropped.
func splitDocuments(lines []physLine) [][]physLine {
	var docs [][]physLine
	var cur []physLine
	for _, ln := range lines {
		trimmed := strings.TrimRight(ln.text, " \t")
		switch {
		case trimmed == "---" || strings.HasPrefix(trimmed, "--- ") || strings.HasPrefix(trimmed, "---\t"):
			docs = append(docs, cur)
			cur = nil
			rest := strings.TrimLeft(strings.TrimPrefix(trimmed, "---"), " \t")
			if rest != "" {
				cur = append(cur, physLine{text: rest, no: ln.no})
			}
		case trimmed == "...":
			docs = append(docs, cur)
			cur = nil
		case strings.HasPrefix(trimmed, "%"):
			// YAML directive: ignored.
		default:
			cur = append(cur, ln)
		}
	}
	docs = append(docs, cur)
	return docs
}

// parser walks the physical lines of a single document.
type parser struct {
	lines []physLine
	pos   int
}

// lineTok is a structural (non-blank, non-comment) line: its tab-indent depth
// and the content that follows the indentation.
type lineTok struct {
	indent  int
	content string
	no      int
	idx     int // index into parser.lines
}

// hasContent reports whether the document holds any structural line.
func (p *parser) hasContent() bool {
	for _, ln := range p.lines {
		_, _, blank, err := measure(ln)
		if err != nil {
			return true // surface the error during real parsing
		}
		if blank {
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(ln.text, "\t"), "#") {
			continue
		}
		return true
	}
	return false
}

// measure splits a physical line into its structural indentation depth and the
// content that follows. Depth is the number of leading TAB characters and
// nothing else. Spaces are permitted only as ALIGNMENT after the tabs -- for
// example to line a mapping body up past a "- " marker -- and never count
// toward depth. Two situations are errors: leading spaces with no preceding tab
// (spaces masquerading as indentation), and a tab that comes after alignment
// spaces (indent first, then align).
func measure(ln physLine) (indent int, content string, blank bool, err error) {
	t := ln.text
	i := 0
	for i < len(t) && t[i] == '\t' {
		i++
	}
	if strings.TrimRight(t[i:], " \t") == "" {
		// Nothing but whitespace after the tabs: a blank line.
		return i, "", true, nil
	}
	j := i
	for j < len(t) && t[j] == ' ' {
		j++
	}
	if i == 0 && j > 0 {
		return 0, "", false, errorf(ln.no, 1,
			"spaces cannot be used for indentation; indent with tabs (spaces only align after a tab)")
	}
	if j < len(t) && t[j] == '\t' {
		return 0, "", false, errorf(ln.no, j+1,
			"tab after spaces; indent with tabs first, then align with spaces")
	}
	return i, t[j:], false, nil
}

// peek returns the next structural line without consuming it, skipping blank
// and whole-line comment lines.
func (p *parser) peek() (*lineTok, error) {
	for i := p.pos; i < len(p.lines); i++ {
		indent, content, blank, err := measure(p.lines[i])
		if err != nil {
			return nil, err
		}
		if blank {
			continue
		}
		if strings.HasPrefix(content, "#") {
			continue
		}
		return &lineTok{indent: indent, content: content, no: p.lines[i].no, idx: i}, nil
	}
	return nil, nil
}

// next returns and consumes the next structural line.
func (p *parser) next() (*lineTok, error) {
	tok, err := p.peek()
	if err != nil || tok == nil {
		return tok, err
	}
	p.pos = tok.idx + 1
	return tok, nil
}

func (p *parser) parseDocument() (any, error) {
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, nil
	}
	if tok.indent != 0 {
		return nil, errorf(tok.no, tok.indent+1, "document must start at the left margin (no leading tabs)")
	}
	v, err := p.parseNode(0)
	if err != nil {
		return nil, err
	}
	// Anything left at this point is mis-indented or stray content.
	if leftover, err := p.peek(); err != nil {
		return nil, err
	} else if leftover != nil {
		return nil, errorf(leftover.no, leftover.indent+1, "unexpected content; check indentation")
	}
	return v, nil
}

// parseNode parses the block (mapping, sequence, or lone scalar) that begins at
// the given indent level.
func (p *parser) parseNode(indent int) (any, error) {
	tok, err := p.peek()
	if err != nil || tok == nil {
		return nil, err
	}
	if isSeqMarker(tok.content) {
		return p.parseSequence(indent)
	}
	if _, _, ok, kerr := splitMapping(tok); kerr == nil && ok {
		return p.parseMapping(indent)
	} else if kerr != nil {
		return nil, kerr
	}
	// A lone scalar value standing on its own line.
	p.next()
	if isBlockScalarHeader(tok.content) {
		return p.parseBlockScalar(tok.content, indent, tok.no)
	}
	return resolveScalar(tok.content, tok.no)
}

func (p *parser) parseMapping(indent int) (any, error) {
	m := map[string]any{}
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok == nil || tok.indent < indent {
			break
		}
		if tok.indent > indent {
			return nil, errorf(tok.no, tok.indent+1, "unexpected indentation in mapping")
		}
		if isSeqMarker(tok.content) {
			// A dash at this depth ends the mapping (it belongs to an enclosing
			// sequence); the caller decides whether that is valid here.
			break
		}
		key, val, ok, err := splitMapping(tok)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errorf(tok.no, indent+1, "expected a \"key: value\" mapping entry")
		}
		if _, dup := m[key]; dup {
			return nil, errorf(tok.no, indent+1, "duplicate mapping key %q", key)
		}
		p.next()
		value, err := p.parseValue(val, indent, tok.no)
		if err != nil {
			return nil, err
		}
		m[key] = value
	}
	return m, nil
}

func (p *parser) parseSequence(indent int) (any, error) {
	s := []any{}
	for {
		tok, err := p.peek()
		if err != nil {
			return nil, err
		}
		if tok == nil || tok.indent < indent {
			break
		}
		if tok.indent > indent {
			return nil, errorf(tok.no, tok.indent+1, "unexpected indentation in sequence")
		}
		if !isSeqMarker(tok.content) {
			break
		}
		p.next() // consume the dash line
		inline := strings.TrimLeft(tok.content[1:], " \t")
		body, err := p.collectItemBody(inline, indent, tok.no)
		if err != nil {
			return nil, err
		}
		v, err := parseItemBody(body, indent)
		if err != nil {
			return nil, err
		}
		s = append(s, v)
	}
	return s, nil
}

// collectItemBody gathers the physical lines making up one sequence item: an
// optional synthetic line carrying the content written inline after the dash,
// followed by every line that belongs to the item. A line belongs when it is
// indented deeper than the dash, or sits at the same tab depth but is not itself
// a new dash -- those same-depth lines are the item's mapping body, aligned past
// the marker with spaces ("tabs for indentation, spaces for alignment").
func (p *parser) collectItemBody(inline string, indent, no int) ([]physLine, error) {
	var body []physLine
	if inline != "" {
		body = append(body, physLine{text: strings.Repeat("\t", indent) + inline, no: no})
	}
	for p.pos < len(p.lines) {
		ln := p.lines[p.pos]
		lineIndent, content, blank, err := measure(ln)
		if err != nil {
			return nil, err
		}
		if blank || strings.HasPrefix(content, "#") {
			body = append(body, ln)
			p.pos++
			continue
		}
		if lineIndent < indent {
			break
		}
		if lineIndent == indent && isSeqMarker(content) {
			break // the next item in this sequence
		}
		body = append(body, ln)
		p.pos++
	}
	return body, nil
}

// parseItemBody parses the collected lines of one sequence item as a single
// node. An item with no body is null. The node is parsed at the depth of its
// first line, which may equal the dash's depth (aligned form) or be deeper.
func parseItemBody(body []physLine, indent int) (any, error) {
	sub := &parser{lines: body}
	tok, err := sub.peek()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, nil // a bare "-" with nothing after it
	}
	v, err := sub.parseNode(tok.indent)
	if err != nil {
		return nil, err
	}
	leftover, err := sub.peek()
	if err != nil {
		return nil, err
	}
	if leftover != nil {
		return nil, errorf(leftover.no, leftover.indent+1, "unexpected content in sequence item; check indentation")
	}
	return v, nil
}

// parseValue resolves the value attached to a "key:" (or an inline "- key:")
// given the inline text after the colon. When the inline text is empty the
// value is the block indented one or more tabs deeper, or null.
func (p *parser) parseValue(inline string, parentIndent, no int) (any, error) {
	if inline == "" {
		child, err := p.peek()
		if err != nil {
			return nil, err
		}
		if child != nil && child.indent > parentIndent {
			return p.parseNode(child.indent)
		}
		return nil, nil
	}
	if isBlockScalarHeader(inline) {
		return p.parseBlockScalar(inline, parentIndent, no)
	}
	return resolveScalar(inline, no)
}

// isSeqMarker reports whether content begins a sequence item: a "-" that is
// either the whole content or immediately followed by whitespace.
func isSeqMarker(content string) bool {
	if content == "" || content[0] != '-' {
		return false
	}
	return len(content) == 1 || content[1] == ' ' || content[1] == '\t'
}

// splitMapping detects a "key: value" (or "key:") entry. The separator is the
// first colon that is followed by whitespace or end-of-content and lies outside
// quotes and flow brackets. The returned val is the trimmed inline text after
// the colon ("" when the colon ends the line). ok is false when no separator is
// present (the content is a bare scalar, not a mapping entry).
func splitMapping(tok *lineTok) (key, val string, ok bool, err error) {
	s := tok.content
	var inSingle, inDouble bool
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			}
		case inDouble:
			if c == '\\' {
				i++
			} else if c == '"' {
				inDouble = false
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			if depth > 0 {
				depth--
			}
		case c == ':' && depth == 0:
			if i+1 == len(s) || s[i+1] == ' ' || s[i+1] == '\t' {
				rawKey := strings.TrimSpace(s[:i])
				k, kerr := unquoteScalar(rawKey, tok.no)
				if kerr != nil {
					return "", "", false, kerr
				}
				return k, strings.TrimSpace(s[i+1:]), true, nil
			}
		}
	}
	return "", "", false, nil
}
