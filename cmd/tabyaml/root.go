package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tabyaml",
	Short: "Work with tab-indented YAML (indentation is tabs, never spaces)",
	Long: `tabyaml parses and emits YAML that uses tabs, and only tabs, for
indentation. A space found in the indentation region of a line is a syntax
error; spaces remain legal inside values, quotes, and flow collections.

Every subcommand reads from the named file, or from standard input when no
file (or "-") is given.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error. tabyaml's own
// errors already carry a "tabyaml:" prefix; other errors (e.g. missing files)
// are printed as-is.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		msg := err.Error()
		if !strings.HasPrefix(msg, "tabyaml:") {
			msg = "tabyaml: " + msg
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
