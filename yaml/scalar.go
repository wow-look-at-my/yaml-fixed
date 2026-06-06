package yaml

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// resolveScalar turns a single line of inline value text into a Go value. It
// dispatches between flow collections, quoted scalars, and plain scalars, and
// strips trailing comments from plain values.
func resolveScalar(s string, no int) (any, error) {
	t := strings.TrimLeft(s, " \t")
	if t == "" {
		return nil, nil
	}
	switch t[0] {
	case '[', '{':
		return parseFlow(t, no)
	case '\'', '"':
		// Quoted: a comment may follow the closing quote.
		body, after, err := scanQuoted(t, no)
		if err != nil {
			return nil, err
		}
		if rest := strings.TrimLeft(after, " \t"); rest != "" && !strings.HasPrefix(rest, "#") {
			return nil, errorf(no, 0, "unexpected text after quoted scalar: %q", rest)
		}
		return body, nil
	default:
		return resolvePlain(stripComment(s)), nil
	}
}

// stripComment removes a trailing "# ..." comment from plain text. A '#' starts
// a comment only at the beginning of the text or when preceded by whitespace.
func stripComment(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			s = s[:i]
			break
		}
	}
	return strings.TrimRight(s, " \t")
}

var (
	reInt   = regexp.MustCompile(`^[-+]?(0|[1-9][0-9]*)$`)
	reHex   = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)
	reOct   = regexp.MustCompile(`^0o[0-7]+$`)
	reFloat = regexp.MustCompile(`^[-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?$`)
)

// resolvePlain applies the YAML 1.2 core schema to an unquoted, comment-stripped
// scalar, returning nil, bool, int, float64, or string.
func resolvePlain(s string) any {
	switch s {
	case "", "~", "null", "Null", "NULL":
		return nil
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	case ".inf", ".Inf", ".INF", "+.inf", "+.Inf", "+.INF":
		return math.Inf(1)
	case "-.inf", "-.Inf", "-.INF":
		return math.Inf(-1)
	case ".nan", ".NaN", ".NAN":
		return math.NaN()
	}
	switch {
	case reInt.MatchString(s):
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return int(v)
		}
	case reHex.MatchString(s):
		if v, err := strconv.ParseInt(s[2:], 16, 64); err == nil {
			return int(v)
		}
	case reOct.MatchString(s):
		if v, err := strconv.ParseInt(s[2:], 8, 64); err == nil {
			return int(v)
		}
	case reFloat.MatchString(s) && strings.ContainsAny(s, ".eE"):
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return s
}

// unquoteScalar resolves a possibly-quoted token to its string value. An
// unquoted token is returned unchanged.
func unquoteScalar(s string, no int) (string, error) {
	if s == "" {
		return "", nil
	}
	if s[0] != '\'' && s[0] != '"' {
		return s, nil
	}
	body, after, err := scanQuoted(s, no)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(after) != "" {
		return "", errorf(no, 0, "unexpected text after quoted scalar: %q", after)
	}
	return body, nil
}

// scanQuoted reads a single- or double-quoted scalar at the start of s,
// returning the decoded string and any remaining text after the closing quote.
func scanQuoted(s string, no int) (body, rest string, err error) {
	switch s[0] {
	case '\'':
		return scanSingle(s, no)
	case '"':
		return scanDouble(s, no)
	default:
		return "", s, nil
	}
}

func scanSingle(s string, no int) (string, string, error) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' { // '' -> '
				b.WriteByte('\'')
				i++
				continue
			}
			return b.String(), s[i+1:], nil
		}
		b.WriteByte(s[i])
	}
	return "", "", errorf(no, 0, "unterminated single-quoted scalar")
}

func scanDouble(s string, no int) (string, string, error) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			return b.String(), s[i+1:], nil
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(s) {
			return "", "", errorf(no, 0, "unterminated double-quoted scalar")
		}
		switch e := s[i]; e {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case '/':
			b.WriteByte('/')
		case '0':
			b.WriteByte(0)
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'v':
			b.WriteByte('\v')
		case 'e':
			b.WriteByte(0x1b)
		case 'x':
			r, ni, err := readHex(s, i+1, 2, no)
			if err != nil {
				return "", "", err
			}
			b.WriteRune(r)
			i = ni
		case 'u':
			r, ni, err := readHex(s, i+1, 4, no)
			if err != nil {
				return "", "", err
			}
			b.WriteRune(r)
			i = ni
		case 'U':
			r, ni, err := readHex(s, i+1, 8, no)
			if err != nil {
				return "", "", err
			}
			b.WriteRune(r)
			i = ni
		default:
			return "", "", errorf(no, 0, "invalid escape sequence \\%c", e)
		}
	}
	return "", "", errorf(no, 0, "unterminated double-quoted scalar")
}

// readHex reads exactly n hex digits starting at index i and returns the rune
// and the index of the last digit consumed.
func readHex(s string, i, n, no int) (rune, int, error) {
	if i+n > len(s) {
		return 0, 0, errorf(no, 0, "truncated unicode escape")
	}
	v, err := strconv.ParseUint(s[i:i+n], 16, 32)
	if err != nil {
		return 0, 0, errorf(no, 0, "invalid unicode escape %q", s[i:i+n])
	}
	r := rune(v)
	if !utf8.ValidRune(r) {
		r = utf8.RuneError
	}
	return r, i + n - 1, nil
}

// isBlockScalarHeader reports whether v is a block-scalar header such as "|",
// ">", "|-", ">+", optionally followed by a comment.
func isBlockScalarHeader(v string) bool {
	if v == "" || (v[0] != '|' && v[0] != '>') {
		return false
	}
	rest := strings.TrimLeft(v[1:], " \t")
	// Allow a single chomping indicator, then only whitespace or a comment.
	if rest != "" && (rest[0] == '-' || rest[0] == '+') {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	return rest == "" || strings.HasPrefix(rest, "#")
}
