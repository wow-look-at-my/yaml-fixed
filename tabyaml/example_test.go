package tabyaml_test

import (
	"fmt"

	"github.com/wow-look-at-my/yaml-fixed/tabyaml"
)

func ExampleUnmarshal() {
	type Config struct {
		Name    string   `yaml:"name"`
		Port    int      `yaml:"port"`
		Modules []string `yaml:"modules"`
	}
	// Note the tabs: indentation is tabs, never spaces.
	src := "name: demo\nport: 9000\nmodules:\n\t- auth\n\t- cache\n"
	var cfg Config
	if err := tabyaml.Unmarshal([]byte(src), &cfg); err != nil {
		panic(err)
	}
	fmt.Printf("%s on %d with %v\n", cfg.Name, cfg.Port, cfg.Modules)
	// Output: demo on 9000 with [auth cache]
}

func ExampleParse_rejectsSpaces() {
	// Two spaces of indentation is a syntax error in tab-YAML.
	_, err := tabyaml.Parse([]byte("server:\n  host: localhost"))
	fmt.Println(err)
	// Output: tabyaml: line 2, column 1: spaces cannot be used for indentation; tab-YAML indents with tabs only
}
