package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "yaml",
	Short: "Work with YAML (indentation is tabs, not spaces)",
	Long: `yaml parses and emits YAML. Indentation is done with tabs; a space in
the indentation region of a line is a syntax error, while spaces remain legal
inside values, quotes, and flow collections.

Every subcommand reads from the named file, or from standard input when no
file (or "-") is given.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error. yaml's own
// errors already carry a "yaml:" prefix; other errors (e.g. missing files)
// are printed as-is.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		msg := err.Error()
		if !strings.HasPrefix(msg, "yaml:") {
			msg = "yaml: " + msg
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

// readInput returns the contents of the file named by the first argument, or of
// standard input when there is no argument (or it is "-").
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) == 1 && args[0] != "-" {
		return os.ReadFile(args[0])
	}
	return io.ReadAll(cmd.InOrStdin())
}
