package yaml

import "strings"

// parseFlow parses a complete flow collection (a "[...]" sequence or "{...}"
// mapping) that makes up the whole value text. Any trailing comment is allowed.
func parseFlow(s string, no int) (any, error) {
	f := &flow{s: s, no: no}
	f.skipSpace()
	v, err := f.value()
	if err != nil {
		return nil, err
	}
	f.skipSpace()
	if f.i < len(f.s) && !strings.HasPrefix(f.s[f.i:], "#") {
		return nil, errorf(no, 0, "unexpected text after flow value: %q", f.s[f.i:])
	}
	return v, nil
}

// flow is a tiny recursive-descent parser over a single line of flow syntax.
type flow struct {
	s  string
	i  int
	no int
}

func (f *flow) skipSpace() {
	for f.i < len(f.s) {
		switch f.s[f.i] {
		case ' ', '\t', '\n', '\r':
			f.i++
		default:
			return
		}
	}
}

func (f *flow) value() (any, error) {
	if f.i >= len(f.s) {
		return nil, errorf(f.no, 0, "unexpected end of flow value")
	}
	switch f.s[f.i] {
	case '[':
		return f.seq()
	case '{':
		return f.mapping()
	default:
		return f.scalar()
	}
}

func (f *flow) seq() (any, error) {
	f.i++ // consume '['
	out := []any{}
	f.skipSpace()
	if f.i < len(f.s) && f.s[f.i] == ']' {
		f.i++
		return out, nil
	}
	for {
		v, err := f.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		f.skipSpace()
		if f.i >= len(f.s) {
			return nil, errorf(f.no, 0, "unterminated flow sequence")
		}
		switch f.s[f.i] {
		case ',':
			f.i++
			f.skipSpace()
			if f.i < len(f.s) && f.s[f.i] == ']' { // trailing comma
				f.i++
				return out, nil
			}
		case ']':
			f.i++
			return out, nil
		default:
			return nil, errorf(f.no, 0, "expected ',' or ']' in flow sequence")
		}
	}
}

func (f *flow) mapping() (any, error) {
	f.i++ // consume '{'
	out := map[string]any{}
	f.skipSpace()
	if f.i < len(f.s) && f.s[f.i] == '}' {
		f.i++
		return out, nil
	}
	for {
		f.skipSpace()
		key, err := f.scalarString()
		if err != nil {
			return nil, err
		}
		f.skipSpace()
		var val any
		if f.i < len(f.s) && f.s[f.i] == ':' {
			f.i++
			f.skipSpace()
			if f.i < len(f.s) && f.s[f.i] != ',' && f.s[f.i] != '}' {
				val, err = f.value()
				if err != nil {
					return nil, err
				}
			}
		}
		if _, dup := out[key]; dup {
			return nil, errorf(f.no, 0, "duplicate mapping key %q in flow mapping", key)
		}
		out[key] = val
		f.skipSpace()
		if f.i >= len(f.s) {
			return nil, errorf(f.no, 0, "unterminated flow mapping")
		}
		switch f.s[f.i] {
		case ',':
			f.i++
			f.skipSpace()
			if f.i < len(f.s) && f.s[f.i] == '}' { // trailing comma
				f.i++
				return out, nil
			}
		case '}':
			f.i++
			return out, nil
		default:
			return nil, errorf(f.no, 0, "expected ',' or '}' in flow mapping")
		}
	}
}

// scalar parses a flow scalar and resolves its type.
func (f *flow) scalar() (any, error) {
	if f.i < len(f.s) && (f.s[f.i] == '\'' || f.s[f.i] == '"') {
		body, rest, err := scanQuoted(f.s[f.i:], f.no)
		if err != nil {
			return nil, err
		}
		f.i = len(f.s) - len(rest)
		return body, nil
	}
	return resolvePlain(f.plainToken()), nil
}

// scalarString parses a flow scalar but always returns it as a string, for use
// as a mapping key.
func (f *flow) scalarString() (string, error) {
	if f.i < len(f.s) && (f.s[f.i] == '\'' || f.s[f.i] == '"') {
		body, rest, err := scanQuoted(f.s[f.i:], f.no)
		if err != nil {
			return "", err
		}
		f.i = len(f.s) - len(rest)
		return body, nil
	}
	return f.plainToken(), nil
}

// plainToken reads an unquoted scalar up to the next flow delimiter (',', ']',
// '}', a line break, or ':' followed by a separator) and returns it trimmed. A
// line break ends the token so that flow collections may span several lines
// (as pretty-printed JSON does).
func (f *flow) plainToken() string {
	start := f.i
	for f.i < len(f.s) {
		c := f.s[f.i]
		if c == ',' || c == ']' || c == '}' || c == '\n' || c == '\r' {
			break
		}
		if c == ':' && (f.i+1 >= len(f.s) || f.s[f.i+1] == ' ' || f.s[f.i+1] == '\t' || f.s[f.i+1] == ',' || f.s[f.i+1] == ']' || f.s[f.i+1] == '}') {
			break
		}
		f.i++
	}
	return strings.TrimSpace(f.s[start:f.i])
}
