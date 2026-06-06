package main

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/yaml-fixed/yaml"
)

var toJSONCmd = &cobra.Command{
	Use:   "to-json [file]",
	Short: "Convert YAML to JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		v, err := yaml.Parse(data)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	},
}

func init() {
	rootCmd.AddCommand(toJSONCmd)
}
