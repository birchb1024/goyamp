package main

import (
	"flag"
	"fmt"
	"github.com/birchb1024/goyamp"
	"io"
	"io/ioutil"
	"log"
	"os"
)

func helpText(out io.Writer, doOrNotDo bool) {
	if !doOrNotDo {
		return
	}
	usage := `TODO help!`
	fmt.Fprint(out, " [File]\n\n")
	flag.CommandLine.SetOutput(out)
	flag.PrintDefaults()
	flag.CommandLine.SetOutput(nil)
	fmt.Fprintln(out, usage)
	os.Exit(0)
}

func makeBooleanFlag(flagVar *bool, switchName string, desc string) {
	flag.BoolVar(flagVar, switchName, false, desc)
	flag.BoolVar(flagVar, string(switchName[0]), false, desc)
}

func main() {

	var help, debugFlag bool
	var outputFormatVar string

	odf := struct{
		switchString string
		defaul string
		description string
		}{ "output", "yaml", "output format: json"}
		
	flag.StringVar(&outputFormatVar, string(odf.switchString[0]), odf.defaul, odf.description)
	flag.StringVar(&outputFormatVar, odf.switchString, odf.defaul, odf.description)

	makeBooleanFlag(&help, "help", "Print helpful text.")
	makeBooleanFlag(&debugFlag, "debug", "Print execution trace.")

	flag.Parse()
	helpText(os.Stderr, help)

	outFormat := goyamp.YAML 
	switch outputFormatVar {
	case "json": 
		outFormat = goyamp.JSON
	case "yaml": 
		outFormat = goyamp.YAML
	default:
		log.Fatalf("error: unknown output syntax '%v'", outputFormatVar)
	}
	
	if ! debugFlag {
		log.SetOutput(ioutil.Discard)
	}
	log.Printf("output = %#v", outputFormatVar)

	commandArgs := []string{}
	if len(flag.Args()) > 0 {
		commandArgs = flag.Args()[1:]
	}
	engine := goyamp.NewExpander(commandArgs, os.Environ(), os.Stdout, outFormat)

	var err error
	if len(flag.Args()) == 0 {
		err = engine.ExpandStream(os.Stdin, "-")

	} else if flag.Arg(0) == "-" {
		err = engine.ExpandStream(os.Stdin, "-")

	} else {
		err = engine.ExpandFile(flag.Arg(0))
	}
	if err != nil {
		panic(fmt.Sprintf("error: %v", err))
	}
}
