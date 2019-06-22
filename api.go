package goyamp

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func (engine *Expander) init(environment []string, argv []string) {
	//    """
	//    Construct a new  environment of globals.
	//    :return: New global dict
	//    """
	envMap := mapy{}
	for _, pair := range environment {
		kv := strings.Split(pair, "=")
		envMap[stringy(kv[0])] = stringy(kv[1])
	}
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
		fmt.Fprintf(os.Stderr, "%v", err)
	}
	engine.globals = &env{
		engine: engine,
		parent: nil,
		bind: map[string]yamly{
			"argv":        classify(argv),
			"env":         envMap,
			"__VERSION__": stringy(Version),
			"__DIR__":     stringy(pwd),
		},
	}
	addBuiltinsToEnv(engine.globals)
}

// Syntax holds different output format syntaxes
type Syntax int

// Constants for YAML and JSON syntax
const (
	YAML Syntax = iota + 0
	JSON
)

//
// NewExpander creates a Goyamp macro expansion engine.
func NewExpander(commandArgs []string, environment []string, ow io.Writer, format Syntax) Expander {

	ex := Expander{
		globals:   nil,
		output:    ow,
		outFormat: format,
	}

	ex.init(environment, commandArgs)
	return ex
}

// ExpandStream reads a stream of YAML and expands it.
func (engine *Expander) ExpandStream(input io.Reader, filename string) error {

	engine.globals.bind["__FILE__"] = stringy(filename)
	return expandStream(input, filename, engine.globals)
}

// ExpandFile reads a file of YAML given a path
func (engine *Expander) ExpandFile(filename string) error {

	engine.globals.bind["__FILE__"] = stringy(filename)
	return expandFile(filename, engine.globals)
}
