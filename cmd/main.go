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

func makeBooleanFlag(flagVar *bool, swich string, desc string) {
	flag.BoolVar(flagVar, swich, false, desc)
	flag.BoolVar(flagVar, string(swich[0]), false, desc)
}

func main() {

	var help, debugFlag bool

	makeBooleanFlag(&help, "help", "Print helpful text.")
	makeBooleanFlag(&debugFlag, "debug", "Print execution trace.")

	flag.Parse()
	helpText(os.Stderr, help)
	if ! debugFlag {
		log.SetOutput(ioutil.Discard)
	}

	commandArgs := []string{}
	if len(flag.Args()) > 0 {
		commandArgs = flag.Args()[1:]
	}
	engine := goyamp.NewExpander(commandArgs, os.Environ(), os.Stdout)

	var err error
	if len(flag.Args()) == 0 {
		err = engine.ExpandStream(os.Stdin, "-")

	} else if flag.Arg(0) == "-" {
		err = engine.ExpandStream(os.Stdin, "-")

	} else {
		err = engine.ExpandFile(flag.Arg(0))
	}
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
