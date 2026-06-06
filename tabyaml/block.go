package tabyaml

import "strings"

// parseBlockScalar consumes the lines of a literal ("|") or folded (">") block
// scalar whose header appeared after a "key:" or "- ". parentIndent is the tab
// depth of the line carrying the header; content lines must be indented deeper.
func (p *parser) parseBlockScalar(header string, parentIndent, no int) (any, error) {
	style := header[0] // '|' or '>'
	chomp := byte(0)   // 0 = clip (default), '-' = strip, '+' = keep
	for _, c := range strings.TrimSpace(header[1:]) {
		switch {
		case c == '-' || c == '+':
			if chomp != 0 {
				return nil, errorf(no, 0, "block scalar has more than one chomping indicator")
			}
			chomp = byte(c)
		case c == '#':
			// start of a trailing comment; stop scanning the header
			goto collect
		case c == ' ' || c == '\t':
			// ignore
		default:
			return nil, errorf(no, 0, "invalid block scalar header %q", header)
		}
	}

collect:
	base := -1
	var lines []string
	for p.pos < len(p.lines) {
		raw := p.lines[p.pos].text
		if strings.TrimSpace(raw) == "" {
			lines = append(lines, "")
			p.pos++
			continue
		}
		tabs := leadingTabs(raw)
		if base == -1 {
			if tabs <= parentIndent {
				break // no content: empty block scalar
			}
			base = tabs
		}
		if tabs < base {
			break
		}
		lines = append(lines, raw[base:])
		p.pos++
	}

	var core string
	if style == '|' {
		core = literal(lines)
	} else {
		core = folded(lines)
	}
	return applyChomp(core, lines, chomp), nil
}

func leadingTabs(s string) int {
	i := 0
	for i < len(s) && s[i] == '\t' {
		i++
	}
	return i
}

// literal joins content lines verbatim, dropping trailing blank lines (their
// reintroduction is governed by chomping).
func literal(lines []string) string {
	n := trimTrailingBlanks(lines)
	return strings.Join(lines[:n], "\n")
}

// folded applies flow folding: line breaks between adjacent non-empty,
// equally-indented lines become single spaces, blank lines become newlines, and
// "more indented" lines (extra leading whitespace) keep their breaks.
func folded(lines []string) string {
	n := trimTrailingBlanks(lines)
	var b strings.Builder
	started := false
	pendingBlanks := 0
	prevMore := false
	for _, ln := range lines[:n] {
		if ln == "" {
			pendingBlanks++
			continue
		}
		more := len(ln) > 0 && (ln[0] == ' ' || ln[0] == '\t')
		switch {
		case !started:
			started = true
		case pendingBlanks > 0:
			b.WriteString(strings.Repeat("\n", pendingBlanks))
		case prevMore || more:
			b.WriteByte('\n')
		default:
			b.WriteByte(' ')
		}
		b.WriteString(ln)
		pendingBlanks = 0
		prevMore = more
	}
	return b.String()
}

// applyChomp adds the trailing newline(s) required by the chomping indicator.
func applyChomp(core string, lines []string, chomp byte) string {
	if core == "" {
		if chomp == '+' {
			return strings.Repeat("\n", len(lines))
		}
		return ""
	}
	switch chomp {
	case '-': // strip: no trailing newline
		return core
	case '+': // keep: final newline plus every trailing blank line
		return core + "\n" + strings.Repeat("\n", trailingBlankCount(lines))
	default: // clip: exactly one trailing newline
		return core + "\n"
	}
}

func trimTrailingBlanks(lines []string) int {
	n := len(lines)
	for n > 0 && lines[n-1] == "" {
		n--
	}
	return n
}

func trailingBlankCount(lines []string) int {
	return len(lines) - trimTrailingBlanks(lines)
}
