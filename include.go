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

func include_builtin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    Sequentially expand a list of YAML files in the current environment.
	//    return: None
	//    """
	log.Printf("include_builtin: %v %v\n", tree, args)
	if _, nok := args.(nily); nok {
		return args
	}
	switch arg := args.(type) {
	case seqy:
		for _, x := range arg {
			maybefile := x.expand(bindings)
			if filename, ok := maybefile.(stringy); ok {
				err := expand_file(string(filename), bindings)
				if err != nil {
					log.Fatalf("%v", err)
				}
			} else {
				panic(fmt.Sprintf("ERROR: include was expecting string filename, got %v for %v in %v", maybefile, x, tree))
			}
		}
	case stringy:
		err := expand_file(string(arg), bindings)
		if err != nil {
			log.Fatalf("%v", err)
		}
	default:
		panic(fmt.Sprintf("ERROR: include was expecting list of filenames, got %#v", args))
	}
	return nily{}
}

func loader(tree yamly, args yamly, bindings *env) (yamly, error) {
	log.Printf("load_builtin %v %v", tree, args)
	//    """
	//    Read a file of data, no macro expansions.
	//    :return: the data as read
	//    """
	// TODO   validate_params(tree, {'': None}, args, '')
	//    return expand_file(args, bindings, expandafterload=False)
	//
	filename, ok := args.(stringy)
	if !ok {
		panic(fmt.Sprintf("ERROR: load was expecting a filename, got '%v'", args))
	}

	// TOD DRY ... cf include_builtin
	var current_dir, path string
	var err error
	current_file, ok := bindings.lookup(stringy("__PATH__"))
	log.Printf("__PATH__ => %v\n", current_file)
	if !ok {
		pwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		current_dir = pwd
	} else {
		current_dir = filepath.Dir(string(current_file.(stringy)))
	}
	if strings.HasPrefix(string(filename), "/") || string(filename) == "-" {
		path = string(filename)
	} else {
		path, err = filepath.Abs(filepath.Join(current_dir, string(filename))) // resolve relative paths
		if err != nil {
			return nil, err
		}
	}
	log.Printf("load_builtin path %v", path)
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
		log.Printf("load_builtin doc %v", doc)
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

func load_builtin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("load_builtin %v %v", tree, args)
	result, err := loader(tree, args, bindings)
	if err != nil {
		panic(fmt.Sprintf("ERROR: load_builtin '%v' %v", args, err))
	}
	return result
}
