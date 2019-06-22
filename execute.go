package goyamp

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os/exec"
	"strings"
	"encoding/json"
)

func argString(tree yamly, args mapy, name string, defaul string) (string) {
	a, ok := args[stringy(name)] 
	if !ok {
		return defaul
	}
	val, ok := a.(stringy) 
	if !ok {
		panic (fmt.Sprintf("'%#v' %v is not a string in '%v'", a, name, tree))
	}
	return string(val)
}

func executeBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("exec: %v", args)
	//
	//    TODO This is all to do...
	//
	validResponseType := map[string]bool{"lines": true, "yaml": true,"json": true, "string": true}
	assertSingleKey(tree)

	var command, directory string
	responseType := "lines"
	requestType := "lines"
	var request yamly = nily{}
	environment := []string{} 
	arguments := []string{}

	
	switch args := args.(type) {
	case stringy:
		str := string(args)
		command = strings.Split(str, " ")[0]
		arguments = strings.Split(str, " ")[1:]
	case mapy:
		assertKeys(map[string]bool{"command": true, "args": false, "environment": false, "directory": false, "request": false, "request-type": false, "response-type": false, }, args)
		command = argString(tree, args, "command", "")
		
		a, aok := args[stringy("args")]
		if aok {
			clst, ok := a.(seqy)
			if ok {
				for _, v := range clst {
					arguments = append(arguments, v.String())
				}
			}
		}

		directory = argString(tree, args, "directory", "")
		requestType = argString(tree, args, "request-type", "lines")
		responseType = argString(tree, args, "response-type", "lines")
		if _, ok := validResponseType[responseType] ; !ok {
			panic(fmt.Sprintf("'%v' is not a valid response-type", responseType))
		}
		
		if envi, eok := args[stringy("environment")]; eok {
			switch envi.(type) {
			case mapy:
				 for k, v := range envi.(mapy) {
				 	environment = append(environment, k.String() + "=" + v.String())
				 }
			default:
				panic(fmt.Sprintf("'%#v' is not a environment map", envi))
			}
		}
		request, _ = args[stringy("request")]

	default:
		panic(fmt.Sprintf("execute args are not string or map %v", args))
	}
	
	log.Printf("execute: '%v' '%v' '%v' '%v' '%v' '%v'", command, directory, environment, requestType, responseType, request)
	if command == "" {
		panic(fmt.Sprintf("execute has no string command %v", args))
	}

	cmd := exec.Command(command, arguments...)
	cmd.Dir = directory
	cmd.Env = environment
	response, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			panic(fmt.Sprintf("%v %v", err.Error(), string(err.Stderr)))
		}
		panic(err)
	}
	
	// Process the response from the sub-process
	//
	responsestr := string(response)
	log.Printf("exec: response %#v", responsestr)
	switch responseType {
	case "lines":
		lineslice := strings.Split(responsestr, "\n")
		result := seqy{}
		for i, l := range lineslice {
			if i == len(lineslice) - 1 && l == "" {
				;			
			} else {
				result = append(result, stringy(l))
			}
		}
		return result
	case "yaml":
		decoder := yaml.NewDecoder(bytes.NewReader(response))
		var doc interface{}
		err = decoder.Decode(&doc)
		if err != nil && err != io.EOF {
			panic(fmt.Sprintf("execute response was not YAML '%v'", err))
		}
		return classify(doc)
	case "json":
		var doc interface{}
		err := json.Unmarshal(response, &doc)
		if err != nil {
			panic(fmt.Sprintf("execute response was not JSON '%v'", err))
		}
		return classify(doc)
	case "string":
		return stringy(responsestr)
	default:
		panic(fmt.Sprintf("execute unknown response type '%v'", responseType))
	}
}
