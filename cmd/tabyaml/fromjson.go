package main

import (
	"bytes"
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/wow-look-at-my/yaml-fixed/tabyaml"
)

var fromJSONCmd = &cobra.Command{
	Use:   "from-json [file]",
	Short: "Convert JSON to tab-YAML",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return err
		}
		out, err := tabyaml.Marshal(normalizeJSON(v))
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(out)
		return err
	},
}

// normalizeJSON converts the json.Number values produced by a number-preserving
// decoder into the int/float64 values that tabyaml.Marshal understands, so that
// JSON integers do not turn into "1.0"-style floats.
func normalizeJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, e := range t {
			m[k] = normalizeJSON(e)
		}
		return m
	case []any:
		for i, e := range t {
			t[i] = normalizeJSON(e)
		}
		return t
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	default:
		return v
	}
}

func init() {
	rootCmd.AddCommand(fromJSONCmd)
}
