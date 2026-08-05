package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	yamlfixed "github.com/wow-look-at-my/yaml-fixed/yaml"
	yamlv3 "gopkg.in/yaml.v3"
)

var migrateWrite bool

var migrateCmd = &cobra.Command{
	Use:   "migrate [file]",
	Short: "Reindent a space-indented YAML file to this package's tab convention",
	Long: `migrate reads a YAML document written in the conventional, space-indented
style and re-emits it using this package's tab-indented form. Unlike fmt,
which requires input already in this package's dialect, migrate accepts
any indentation width and mix -- it exists to convert an existing file once,
not to validate it.

Mapping key order is preserved from the source. Anchors, aliases, and merge
keys are not supported by this package and are rejected with an error.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := readInput(cmd, args)
		if err != nil {
			return err
		}
		out, err := migrate(data)
		if err != nil {
			return err
		}
		if migrateWrite {
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
	migrateCmd.Flags().BoolVarP(&migrateWrite, "write", "w", false, "write the result back to the file instead of stdout")
	rootCmd.AddCommand(migrateCmd)
}

// migrate parses data with a standard, indentation-width-tolerant YAML reader
// and re-marshals every document through yaml-fixed, converting indentation
// to tabs and normalizing formatting while preserving mapping key order and
// scalar content exactly.
func migrate(data []byte) ([]byte, error) {
	dec := yamlv3.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	first := true
	for {
		var doc yamlv3.Node
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parsing source YAML: %w", err)
		}
		v, err := nodeToValue(documentRoot(&doc))
		if err != nil {
			return nil, err
		}
		b, err := yamlfixed.Marshal(v)
		if err != nil {
			return nil, err
		}
		if !first {
			out.WriteString("---\n")
		}
		first = false
		out.Write(b)
	}
	return out.Bytes(), nil
}

// documentRoot unwraps a decoded document node to its actual content node.
func documentRoot(n *yamlv3.Node) *yamlv3.Node {
	if n.Kind == yamlv3.DocumentNode && len(n.Content) == 1 {
		return n.Content[0]
	}
	return n
}

// nodeToValue converts a gopkg.in/yaml.v3 node tree into the generic value
// model yaml-fixed's own Marshal understands, preserving mapping key order via
// *yamlfixed.Map instead of collapsing into a plain, order-free Go map.
func nodeToValue(n *yamlv3.Node) (any, error) {
	if n.Anchor != "" {
		return nil, fmt.Errorf("line %d: anchors are not supported", n.Line)
	}
	switch n.Kind {
	case yamlv3.ScalarNode:
		return scalarNodeToValue(n)
	case yamlv3.SequenceNode:
		out := make([]any, len(n.Content))
		for i, c := range n.Content {
			v, err := nodeToValue(c)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	case yamlv3.MappingNode:
		return mappingNodeToValue(n)
	case yamlv3.AliasNode:
		return nil, fmt.Errorf("line %d: aliases are not supported", n.Line)
	default:
		return nil, fmt.Errorf("line %d: unsupported node kind", n.Line)
	}
}

func mappingNodeToValue(n *yamlv3.Node) (any, error) {
	m := &yamlfixed.Map{Values: map[string]any{}}
	for i := 0; i+1 < len(n.Content); i += 2 {
		keyNode, valNode := n.Content[i], n.Content[i+1]
		if keyNode.Tag == "!!merge" {
			return nil, fmt.Errorf("line %d: merge keys are not supported", keyNode.Line)
		}
		key, err := scalarNodeToValue(keyNode)
		if err != nil {
			return nil, err
		}
		keyStr, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("line %d: mapping key %v is not a string", keyNode.Line, key)
		}
		if _, dup := m.Values[keyStr]; dup {
			return nil, fmt.Errorf("line %d: duplicate mapping key %q", keyNode.Line, keyStr)
		}
		val, err := nodeToValue(valNode)
		if err != nil {
			return nil, err
		}
		m.Keys = append(m.Keys, keyStr)
		m.Values[keyStr] = val
	}
	return m, nil
}

// scalarNodeToValue resolves a scalar node's YAML 1.1 core-schema type
// (gopkg.in/yaml.v3's resolver) into yaml-fixed's own generic value universe.
// Anything outside null/bool/int/float -- a timestamp, binary, or other
// custom-tagged scalar -- is carried through as its literal string content
// rather than as a Go type yaml-fixed's Marshal cannot represent.
func scalarNodeToValue(n *yamlv3.Node) (any, error) {
	if n.Anchor != "" {
		return nil, fmt.Errorf("line %d: anchors are not supported", n.Line)
	}
	switch n.Tag {
	case "!!null":
		return nil, nil
	case "!!bool":
		var b bool
		if err := n.Decode(&b); err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Line, err)
		}
		return b, nil
	case "!!int":
		var i int64
		if err := n.Decode(&i); err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Line, err)
		}
		return i, nil
	case "!!float":
		var f float64
		if err := n.Decode(&f); err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Line, err)
		}
		return f, nil
	default:
		return n.Value, nil
	}
}
