package internal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func (engine *Expander) init(environment []string, argv []string) {
	//    """
	//    Construct a new  environment of globals.
	//    :return: New global dict
	//    """
	envMap := mapy{}
	for _, pair := range environment {
		kv := strings.SplitN(pair, "=", 2)
		envMap[stringy(kv[0])] = stringy(kv[1])
	}
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "."
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
	}
	engine.globals = &env{
		engine: engine,
		parent: nil,
		bind: map[string]yamly{
			"argv":        classify(argv),
			"env":         envMap,
			"__VERSION__": stringy(engine.version),
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
	LINES
)

// NewExpander creates a Goyamp macro expansion engine.
func NewExpander(commandArgs []string, environment []string, ow io.Writer, format Syntax, version string) Expander {

	ex := Expander{
		globals:   nil,
		output:    ow,
		outFormat: format,
		version:   version,
	}

	ex.init(environment, commandArgs)
	return ex
}

// ExpandStream reads a stream of YAML and expands it.
func (engine *Expander) ExpandStream(input io.Reader, filename string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%+v", r))
		}
	}()
	engine.globals.bind["__FILE__"] = stringy(filename)
	return expandStream(input, filename, engine.globals)
}

// ExpandFile reads a file of YAML given a path
func (engine *Expander) ExpandFile(filename string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%+v", r))
		}
	}()
	engine.globals.bind["__FILE__"] = stringy(filename)
	return expandFile(filename, engine.globals)
}
