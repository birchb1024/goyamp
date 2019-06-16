package main

import (
	"bytes"
	"fmt"
	"flag"
	"log"
	"io"
	"os"
	"io/ioutil"
	"path/filepath"
	"testing"
	"github.com/birchb1024/goyamp"
)

func file_runner(output io.Writer, filename string) error {
	engine := goyamp.NewExpander(
		[]string{"A", "B", "C", "D"},
		[]string{"USERNAME=birchb", "USER=birchb"},
		output)
	return engine.ExpandFile(filename)
}

func Test_normalExamples(t *testing.T) {

	normalExampleFiles := []string{
                "alter_keys.yaml",
                "arguments.yaml",
                "caret.yaml",
                "config.yaml",
                "env01.yaml",
                "flatten-repeat.yaml",
                "flatten.yaml",
                "foo.yaml",
                "for.yaml",
                "funny_variables.yaml",
                "ifs.yaml",
                "includes.yaml",
                "issue06.yaml",
                "load_data.yaml",
                "loader.yaml",
                "macros.yaml",
                "multi_define.yaml",
                "quote.yaml",
                "range.yaml",
                "readme.yaml",
                "recursive.yaml",
                "repeat02.yaml",
                "repeat-list-keys.yaml",
                "repeat.yaml",
                "replace.yaml",
                "rookout.yaml",
                "template01.gocd.yaml",
                "undefine.yaml",
                "varargs.yaml",
	}
	runTestFiles("Normal examples", normalExampleFiles, t)
}

func TODO_Test_panicExamples(t *testing.T) {

    defer func() {
        if r := recover(); r == nil {
              t.Errorf("The code did not panic")
        } else {
        	fmt.Printf("The code paniced\n")
        }
    }()

	files := []string{
                "asserts.yaml",
	}
	runTestFiles("Panic examples", files, t)
}


func runTestFiles(name string, fileList []string, t *testing.T) {

	for _, filename := range fileList {
		path := fmt.Sprintf("../examples/%v", filename)
		log := fmt.Sprintf("fixtures/examples/%v.log", filename)
		logPath, _ := filepath.Abs(log)
		t.Run(name + "_" + filename, func(t *testing.T) {
			var result bytes.Buffer
			err := file_runner(&result, path)
			if err != nil {
				t.Error(path, err)
				return
			}
			expected, err := ioutil.ReadFile(log)
			if err != nil {
				t.Error(path, err)
				return
			}
			if result.String() != string(expected) {
				err := ioutil.WriteFile("/tmp/"+ filename + ".log", result.Bytes(), 0644)
				if err != nil {
					panic(err)
				}
				t.Error(fmt.Sprintf("output mismatch:\ndiff /tmp/%v.log %v\n", filename, logPath))
				return
			}
		})
	}
}

func TestMain(m *testing.M) {
	var debugFlag bool
	flag.BoolVar(&debugFlag, "d", false, "output debug strings")
	flag.Parse()
	if !debugFlag {
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}