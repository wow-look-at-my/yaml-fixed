package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/yaml-fixed/yaml"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Check that the input is well-formed YAML",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		if _, err := yaml.ParseAll(data); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
