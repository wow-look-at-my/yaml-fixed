package yaml

import (
	"fmt"
	"os"
)

// Warn is called by the parser to report a non-fatal warning about input it
// accepted but wants to flag. Currently the parser's only warning is that it
// accepted space indentation while consuming a JSON document; see ParseAll. It
// is invoked at most once per Parse or ParseAll call ("once per file").
//
// The default writes to standard error. Replace it to capture warnings in
// tests, route them through a logger, or silence them entirely with
// func(string) {}.
var Warn = func(msg string) {
	fmt.Fprintln(os.Stderr, "yaml: warning: "+msg)
}
