package main

import (
	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/yaml-fixed/yaml"
)

var fromJSONCmd = &cobra.Command{
	Use:   "from-json [file]",
	Short: "Convert JSON to YAML",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		// YAML is a superset of JSON, so the YAML parser reads JSON directly:
		// a JSON document is just a flow collection, parsed regardless of how it
		// is indented (spaces and all, with a one-per-file warning). There is no
		// need for a second, separate JSON parser here -- one parser handles both.
		v, err := yaml.Parse(data)
		if err != nil {
			return err
		}
		out, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(out)
		return err
	},
}

func init() {
	rootCmd.AddCommand(fromJSONCmd)
}
