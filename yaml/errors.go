package yaml

import "fmt"

// SyntaxError describes a problem encountered while scanning or parsing a
// YAML document. Line and Col are 1-based; Col is 0 when a column is not
// meaningful for the error.
type SyntaxError struct {
	Line int
	Col  int
	Msg  string
}

func (e *SyntaxError) Error() string {
	if e.Col > 0 {
		return fmt.Sprintf("yaml: line %d, column %d: %s", e.Line, e.Col, e.Msg)
	}
	if e.Line > 0 {
		return fmt.Sprintf("yaml: line %d: %s", e.Line, e.Msg)
	}
	return "yaml: " + e.Msg
}

func errorf(line, col int, format string, args ...any) *SyntaxError {
	return &SyntaxError{Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

// TypeError is returned by Unmarshal when a parsed value cannot be stored into
// the provided Go target.
type TypeError struct {
	Msg string
}

func (e *TypeError) Error() string { return "yaml: " + e.Msg }

func typeErrorf(format string, args ...any) *TypeError {
	return &TypeError{Msg: fmt.Sprintf(format, args...)}
}
