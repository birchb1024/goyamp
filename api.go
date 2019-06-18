package goyamp

import (
	"io"
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

	engine.globals = &env{
		engine: engine,
		parent: nil,
		bind: map[string]yamly{
			"argv":        classify(argv),
			"env":         envMap,
			"__VERSION__": stringy(Version),
		},
	}
	addBuiltinsToEnv(engine.globals)
}

//
// NewExpander creates a Goyamp macro expansion engine.
func NewExpander(commandArgs []string, environment []string, ow io.Writer) Expander {

	ex := Expander{
		globals: nil,
		output:  ow,
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
