package goyamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

func argString(tree yamly, args mapy, name string, defualt string) string {
	a, ok := args[stringy(name)]
	if !ok {
		return defualt
	}
	val, ok := a.(stringy)
	if !ok {
		panic(fmt.Sprintf("'%#v' %v is not a string in '%v'", a, name, tree))
	}
	return string(val)
}

func executeBuiltin(tree mapy, args yamly, _ *env) yamly {
	log.Printf("exec: %v", args)

	validResponseType := map[string]bool{"lines": true, "yaml": true, "json": true, "string": true}
	validRequestType := map[string]bool{"lines": true, "yaml": true, "json": true, "string": true}
	assertSingleKey(tree)

	var command, directory string
	responseType := "lines"
	requestType := "lines"
	request := []byte{}
	environment := os.Environ()
	arguments := []string{}

	switch args := args.(type) {
	case stringy:
		str := string(args)
		command = strings.Split(str, " ")[0]
		arguments = strings.Split(str, " ")[1:]
		responseType = "string"
	case mapy:
		assertKeys(map[string]bool{"command": true, "args": false, "environment": false, "directory": false, "request": false, "request-type": false, "response-type": false}, args)
		command = argString(tree, args, "command", "")

		a, aok := args[stringy("args")]
		if aok {
			clst, ok := a.(seqy)
			if !ok {
				panic(fmt.Sprintf("execute: '%v' is not a valid args sequence", a))
			} else {
				for _, v := range clst {
					arguments = append(arguments, v.String())
					log.Printf("execute: '%v' '%v' '%v' '%v' '%v' '%v' '%v'", command, arguments, directory, environment, requestType, responseType, string(request))
				}
			}
		}
		log.Printf("execute: '%v' '%v' '%v' '%v' '%v' '%v' '%v'", command, arguments, directory, environment, requestType, responseType, string(request))

		directory = argString(tree, args, "directory", "")
		requestType = argString(tree, args, "request-type", "lines")
		if _, ok := validRequestType[responseType]; !ok {
			panic(fmt.Sprintf("'%v' is not a valid request-type", requestType))
		}
		responseType = argString(tree, args, "response-type", "lines")
		if _, ok := validResponseType[responseType]; !ok {
			panic(fmt.Sprintf("'%v' is not a valid response-type", responseType))
		}

		if envi, eok := args[stringy("environment")]; eok {
			switch envi.(type) {
			case mapy:
				for k, v := range envi.(mapy) {
					environment = append(environment, k.String()+"="+v.String())
				}
			default:
				panic(fmt.Sprintf("'%#v' is not a environment map", envi))
			}
		}

		log.Printf("execute: '%v' '%v' '%v' '%v' '%v' '%v' '%v'", command, arguments, directory, environment, requestType, responseType, string(request))
		if req, eok := args[stringy("request")]; eok {
			switch requestType {
			case "string":
				request = []byte(req.String())
			case "lines":
				switch req.(type) {
				case stringy:
					request = []byte(req.String())
				case seqy:
					var buf bytes.Buffer
					for _, line := range req.(seqy) {
						buf.WriteString(line.String())
						buf.WriteString("\n")
					}
					request = buf.Bytes()
				default:
					panic(fmt.Sprintf("execute: '%v' is not a string or sequence for lines", req))
				}

			case "json":
				var buf bytes.Buffer
				j := req.declassify(JSON)
				jenc := json.NewEncoder(&buf)
				jenc.SetIndent("", "  ")
				err := jenc.Encode(j)
				if err != nil {
					panic(fmt.Sprintf("execute: '%v' could not be encoded as JSON", req))
				}
				request = buf.Bytes()

			case "yaml":
				var buf bytes.Buffer
				y := req.declassify()
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				err := enc.Encode(y)
				if err != nil {
					panic(fmt.Sprintf("execute: '%v' could not be encoded as YAML!", req))
				}
				request = buf.Bytes()

			default:
				panic(fmt.Sprintf("'%v' is not a valid request-type", requestType))
			}
		}

	default:
		panic(fmt.Sprintf("execute args are not string or map %v", args))
	}

	log.Printf("execute: '%v' '%#v' '%v' '%v' '%v' '%v' '%v'", command, arguments, directory, environment, requestType, responseType, string(request))
	if command == "" {
		panic(fmt.Sprintf("execute has no string command %v", args))
	}

	cmd := exec.Command(command, arguments...)
	cmd.Dir = directory
	cmd.Env = environment
	cmd.Stderr = os.Stderr // process stderr goes direct to user stderr
	var responseBuffer bytes.Buffer
	cmd.Stdout = &responseBuffer
	if request != nil {
		cmd.Stdin = bytes.NewReader(request)
	}
	log.Printf("execute: start: '%#v'", cmd)
	err := cmd.Run()
	response := responseBuffer.Bytes()
	log.Printf("execute: done.")
	if err != nil {
		panic(err)
	}

	// Process the response from the sub-process
	//
	responsestr := string(response)
	log.Printf("exec: response %#v", responsestr)
	switch responseType {
	case "string":
		return stringy(strings.TrimSpace(responsestr))
	case "lines":
		lineslice := strings.Split(responsestr, "\n")
		result := seqy{}
		for i, l := range lineslice {
			if i == len(lineslice)-1 && l == "" {

			} else {
				result = append(result, stringy(l))
			}
		}
		return result
	case "json":
		var doc interface{}
		if len(response) == 0 {
			return nily{} // JSON parser can't deal with ""
		}
		err := json.Unmarshal(response, &doc)
		if err != nil {
			panic(fmt.Sprintf("execute response, '%s', was not JSON '%v'", response, err))
		}
		return classify(intify(doc)) // convert floats to ints where possible, because JSON uses only float64 :-(
	case "yaml":
		decoder := yaml.NewDecoder(bytes.NewReader(response))
		var doc interface{}
		err = decoder.Decode(&doc)
		if err != nil && err != io.EOF {
			panic(fmt.Sprintf("execute response '%s', was not YAML '%v'", response, err))
		}
		return classify(doc)
	default:
		panic(fmt.Sprintf("execute unknown response type '%v'", responseType))
	}
}
