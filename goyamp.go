package main

import (
	"flag"
	"fmt"
	"github.com/birchb1024/goyamp/internal"
	"io"
	"log"
	"os"
	"runtime/debug"
	"strings"
)

func helpText(out io.Writer, doOrNotDo bool) {
	if !doOrNotDo {
		return
	}
	usage := `
USAGE:

 $ goyamp [-d|-debug] [-h|-help] [-o|-output yaml|json|lines] [Filename | - ] [arg1..argn]

	-
	Filename:    If the filename is the minus sign '-' or if there are no arguments, Goamp reads YAML from the standard input. 

	arg1-argn:   are passed to the processor in the 'argv' variable.

	-o
	-output:     If the -output option specifies the output format required. The options are yaml, json or lines. the default is YAML.

	-d
	-debug:      prints a trace of internal execution.

	-h
	-help:       Prints this text.
`
	fmt.Fprintln(out, usage)
	os.Exit(0)
}

func makeBooleanFlag(flagVar *bool, switchName string, desc string) {
	flag.BoolVar(flagVar, switchName, false, desc)
	flag.BoolVar(flagVar, string(switchName[0]), false, desc)
}

// Version provides the Git version tag used in the build of the binary
var Version string

func main() {

	version :=  getBuildInfoVersion(Version)

	var help, debugFlag bool
	var outputFormatVar string

	odf := struct {
		switchString string
		defaul       string
		description  string
	}{"output", "yaml", "output format: json/yaml/lines"}

	flag.StringVar(&outputFormatVar, string(odf.switchString[0]), odf.defaul, odf.description)
	flag.StringVar(&outputFormatVar, odf.switchString, odf.defaul, odf.description)

	makeBooleanFlag(&help, "help", "Print helpful text.")
	makeBooleanFlag(&debugFlag, "debug", "Print execution trace.")

	flag.Parse()
	helpText(os.Stderr, help)

	outFormat := internal.YAML
	switch outputFormatVar {
	case "json":
		outFormat = internal.JSON
	case "yaml":
		outFormat = internal.YAML
	case "lines":
		outFormat = internal.LINES
	default:
		log.Fatalf("error: unknown output syntax '%v'", outputFormatVar)
	}

	if !debugFlag {
		log.SetOutput(io.Discard)
	}
	log.Printf("output = %#v", outputFormatVar)

	commandArgs := []string{}
	if len(flag.Args()) > 0 {
		commandArgs = flag.Args()[1:]
	}
	engine := internal.NewExpander(commandArgs, os.Environ(), os.Stdout, outFormat, version)

	var err error
	if len(flag.Args()) == 0 {
		err = engine.ExpandStream(os.Stdin, "-")

	} else if flag.Arg(0) == "-" {
		err = engine.ExpandStream(os.Stdin, "-")

	} else {
		err = engine.ExpandFile(flag.Arg(0))
	}
	if err != nil {
		format := "Error: %v\n"
		if debugFlag {
			format = "Error: %+v\n"
		}
		fmt.Fprintf(os.Stderr, format, err)
		panic(2)
	}
}

func getBuildInfoVersion(ldflagVersion string) string {
	parts := []string{}
	if ldflagVersion != "" {
		parts = append(parts, ldflagVersion)
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return strings.Join(parts, "-")
	}
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			parts = append(parts, kv.Value[:9])
		//case "vcs.time":LastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			parts = append(parts, "dirty")
		}
	}
	return strings.Join(parts, "-")
}
