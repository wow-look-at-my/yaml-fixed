package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/yaml-fixed/tabyaml"
)

var fmtWrite bool

var fmtCmd = &cobra.Command{
	Use:   "fmt [file]",
	Short: "Canonicalise tab-YAML (sort keys, normalise indentation to tabs)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		v, err := tabyaml.Parse(data)
		if err != nil {
			return err
		}
		out, err := tabyaml.Marshal(v)
		if err != nil {
			return err
		}
		if fmtWrite {
			if len(args) != 1 || args[0] == "-" {
				return fmt.Errorf("--write requires a file argument")
			}
			return os.WriteFile(args[0], out, 0o644)
		}
		_, err = cmd.OutOrStdout().Write(out)
		return err
	},
}

func init() {
	fmtCmd.Flags().BoolVarP(&fmtWrite, "write", "w", false, "write the result back to the file instead of stdout")
	rootCmd.AddCommand(fmtCmd)
}
