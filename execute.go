package goyamp

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os/exec"
	"strings"
)

func execute_builtin(tree mapy, args yamly, bindings *env) yamly {
	//
	//    TODO This is all to do...
	//
	// TODO    assert_single_key(tree)
	var command string
	arguments := []string{}
	switch args := args.(type) {
	case stringy:
		str := string(args)
		command = strings.Split(str, " ")[0]
		arguments = strings.Split(str, " ")[1:]
	case mapy:
		c, cok := args[stringy("command")]
		a, aok := args[stringy("args")]
		if cok {
			cstr, ok := c.(stringy)
			if ok {
				command = string(cstr)
			}
		} else {
			panic(fmt.Sprintf("execute cannot exec %v", args))
		}
		if aok {
			clst, ok := a.(seqy)
			if ok {
				for _, v := range clst {

					arguments = append(arguments, string(v.(stringy)))
				}
			} else {
				panic(fmt.Sprintf("execute: '%v' is not a list of strings", a))
			}
		}
	default:
		panic(fmt.Sprintf("execute cannot exec %v", args))
	}
	log.Printf("exec: %v %v", command, arguments)
	cmd := exec.Command(command, arguments...)
	response, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	responsestr := string(response)
	log.Printf("exec: response %#v", responsestr)
	// Attempt to parse YAML/JSON
	decoder := yaml.NewDecoder(bytes.NewReader(response))
	var doc interface{}
	err = decoder.Decode(&doc)
	log.Printf("exec: decoder %#v", err)
	if err != nil && err != io.EOF {
		return stringy(responsestr)
	}
	log.Printf("exec: doc %#v", doc)
	return classify(doc)
}
