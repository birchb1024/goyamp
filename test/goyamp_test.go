package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/birchb1024/goyamp/internal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func fileRunner(output io.Writer, filename string, format internal.Syntax) error {
	engine := internal.NewExpander(
		[]string{"A", "B", "C", "D"},
		[]string{"USERNAME=birchb", "USER=birchb", "TEST_EMBEDDED=x=y,y=5"},
		output,
		format,
		"9.9.9")
	return engine.ExpandFile(filename)
}

func runTestFiles(name string, fileList []string, format internal.Syntax, t *testing.T) {

	typeFlag := ""
	if format == internal.JSON {
		typeFlag = "json."
	}
	for _, filename := range fileList {
		path := fmt.Sprintf("../examples/%v", filename)
		logFile := fmt.Sprintf("fixtures/examples/%v.%vlog", filename, typeFlag)
		logPath, _ := filepath.Abs(logFile)
		t.Run(name+"_"+filename, func(t *testing.T) {
			var result bytes.Buffer
			err := fileRunner(&result, path, format)
			if err != nil {
				t.Error(path, err)
				return
			}
			expected, err := ioutil.ReadFile(logFile)
			if err != nil {
				t.Error(path, err)
				return
			}
			if result.String() != string(expected) {
				err := ioutil.WriteFile("/tmp/"+filename+".logFile", result.Bytes(), 0644)
				if err != nil {
					panic(err)
				}
				t.Error(fmt.Sprintf("output mismatch:\ndiff /tmp/%v.logFile %v\n", filename, logPath))
				return
			}
		})
	}
}

func TestNormalExamples(t *testing.T) {

	normalExampleFiles := []string{
		"alter_keys.yaml",
		"arguments.yaml",
		"caret.yaml",
		"config.yaml",
		"env01.yaml",
		"execute.yaml",
		"flatten-repeat.yaml",
		"flatten.yaml",
		"foo.yaml",
		"for.yaml",
		"funny_variables.yaml",
		"gopherlua.yaml",
		"ifs.yaml",
		"includes.yaml",
		"issue06.yaml",
		"json_numbers.yaml",
		"load_data.yaml",
		"loader.yaml",
		"luadeepmerge-tests.yaml",
		"macros.yaml",
		"macro-argless.yaml",
		"math.yaml",
		"merge.yaml",
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
		"variety.yaml",
	}
	runTestFiles("Normal examples", normalExampleFiles, internal.YAML, t)
}

func TestNormalJSONExamples(t *testing.T) {

	normalExampleFiles := []string{
		"widgets.json",
		"items.json",
		"variety.json",
		"variety.yaml",
		"ifs.yaml",
		"multi_define.yaml",
		"flatten-repeat.yaml",
	}
	runTestFiles("Normal JSON examples", normalExampleFiles, internal.JSON, t)
}

//func TODOTestPanicExamples(t *testing.T) {
//
//	defer func() {
//		if r := recover(); r == nil {
//			t.Errorf("The code did not panic")
//		} else {
//			fmt.Printf("The code paniced\n")
//		}
//	}()
//
//	files := []string{
//		"asserts.yaml",
//	}
//	runTestFiles("Panic examples", files, goyamp.YAML, t)
//}

func TestMain(m *testing.M) {
	var debugFlag bool
	flag.BoolVar(&debugFlag, "d", false, "output debug strings")
	flag.Parse()
	if !debugFlag {
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}
