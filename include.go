package goyamp

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func includeBuiltin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    Sequentially expand a list of YAML files in the current environment.
	//    return: None
	//    """
	log.Printf("includeBuiltin: %v %v\n", tree, args)
	if _, nok := args.(nily); nok {
		return args
	}
	switch arg := args.(type) {
	case seqy:
		for _, x := range arg {
			maybefile := x.expand(bindings)
			if filename, ok := maybefile.(stringy); ok {
				err := expandFile(string(filename), bindings)
				if err != nil {
					log.Fatalf("%v", err)
				}
			} else {
				panic(fmt.Sprintf("ERROR: include was expecting string filename, got %v for %v in %v", maybefile, x, tree))
			}
		}
	case stringy:
		err := expandFile(string(arg), bindings)
		if err != nil {
			log.Fatalf("%v", err)
		}
	default:
		panic(fmt.Sprintf("ERROR: include was expecting list of filenames, got %#v", args))
	}
	return nily{}
}

func loader(tree yamly, args yamly, bindings *env) (yamly, error) {
	log.Printf("loadBuiltin %v %v", tree, args)
	//    """
	//    Read a file of data, no macro expansions.
	//    :return: the data as read
	//    """
	// TODO   validateParams(tree, {'': None}, args, '')
	//    return expandFile(args, bindings, expandafterload=False)
	//
	filename, ok := args.(stringy)
	if !ok {
		panic(fmt.Sprintf("ERROR: load was expecting a filename, got '%v'", args))
	}

	// TOD DRY ... cf includeBuiltin
	var currentDir, path string
	var err error
	currentFile, ok := bindings.lookup(stringy("__PATH__"))
	log.Printf("__PATH__ => %v\n", currentFile)
	if !ok {
		pwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		currentDir = pwd
	} else {
		currentDir = filepath.Dir(string(currentFile.(stringy)))
	}
	if strings.HasPrefix(string(filename), "/") || string(filename) == "-" {
		path = string(filename)
	} else {
		path, err = filepath.Abs(filepath.Join(currentDir, string(filename))) // resolve relative paths
		if err != nil {
			return nil, err
		}
	}
	log.Printf("loadBuiltin path %v", path)
	input, err := os.Open(path)
	if err != nil {
		log.Printf("open => %v %v %v\n", path, input, err)
		return nil, err
	}

	result := seqy{}
	decoder := yaml.NewDecoder(input)
	for {
		var doc interface{}
		err = decoder.Decode(&doc)
		log.Printf("loadBuiltin doc %v", doc)
		if err != nil {
			break
		}
		result = append(result, classify(doc))
	}
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(result) == 1 {
		return result[0], nil
	}
	return result, nil
}

func loadBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("loadBuiltin %v %v", tree, args)
	result, err := loader(tree, args, bindings)
	if err != nil {
		panic(fmt.Sprintf("ERROR: loadBuiltin '%v' %v", args, err))
	}
	return result
}
