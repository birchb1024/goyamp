package goyamp

import (
	"fmt"
	"log"
)

func mapDefine(defs mapy, bindings *env) yamly {
	log.Printf("mapDefine: %v\n", defs)
	definitions := defs.expand(bindings)
	defmap, ok := definitions.(mapy)
	if !ok {
		panic(fmt.Sprintf("Syntax error '%v' not a map from %v", definitions, defs))
	}
	for k, v := range defmap {
		varname, ok := k.(stringy)
		if !ok {
			panic(fmt.Sprintf("define variable name %v must be string in %v", k, defs))
		}
		bindings.bind[string(varname)] = v
	}
	return nily{}
}

func defineBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("defineBuiltin:\n   %#v\n   %#v\n", tree, args)
	//    """
	//    Define one or more variables in the current scope.
	//    :return: None
	//    """
	assertSingleKey(tree)
	todefine, aok := args.(mapy)
	if !aok {
		panic(fmt.Sprintf("define expects a map in %v got %T", tree, args))
	}
	name, nok := todefine[stringy("name")]
	value, vok := todefine[stringy("value")]
	log.Printf("defineBuiltin: nok vok %v %v\n", nok, vok)
	if !nok && !vok {
		return mapDefine(todefine, bindings)
	}
	if !(nok && vok) {
		panic(fmt.Sprintf("Syntax error '%v' missing keyword in %v", args, tree))
	}
	namestr, ok := name.(stringy)
	if !ok {
		panic(fmt.Sprintf("Syntax error '%v' not a string in %v", name, tree))
	}
	newvalue := value.expand(bindings)
	bindings.bind[string(namestr)] = newvalue
	log.Printf("\ndefineBuiltin:\n new value for :  %#v\n value:  %#v\n", namestr, newvalue)
	return nily{}
}

func undefineBuiltin(tree mapy, args yamly, bindin *env) yamly {
	log.Printf("undefineBuiltin:\n   %#v\n   %#v\n", tree, args)
	//    """
	//    Remove binding in the current environment only.
	//    :return: None
	//    """
	assertSingleKey(tree)
	if variable, ok := args.(stringy); ok {
		delete(bindin.bind, string(variable))
		return nily{}
	}
	panic(fmt.Sprintf("Syntax error was expecting string in %v got %v", tree, args))
}

func defmacroBuiltin(tree mapy, args yamly, bindin *env) yamly {
	//    """
	//    Define a new macro.
	//    :return: None
	//    """
	if args == nil {
		panic(fmt.Sprintf("Syntax error empty defmacro %v", tree))
	}
	argst, ok := args.(mapy)
	if !ok {
		panic("Syntax error defmacro expected a map, got %v")
	}
	macroName, ok := argst[stringy("name")]
	if !ok {
		panic(fmt.Sprintf("missing names in %v", tree))
	}
	macroNameStr, ok := macroName.(stringy)
	if !ok {
		panic(fmt.Sprintf("in macro %v is not a string", macroName))
	}

	// Collate arguments
	var macroParams = seqy{} // default is list if no args
	varargs := false
	macrop, ok := argst[stringy("args")]
	if ok {
		switch theparams := macrop.(type) {
		case nily:
		case stringy: //varargs
			macroParams = append(macroParams, theparams)
			varargs = true
		case seqy:
			for _, k := range theparams {
				if paramName, ok := k.(stringy); ok {
					macroParams = append(macroParams, paramName)
				} else {
					panic(fmt.Sprintf("in defmacro args '%#v' is not a string", k))
				}
			}
		default:
			panic(fmt.Sprintf("in defmacro args '%#v' is not a string", macrop))
		}
	}
	macroBody, ok := argst[stringy("value")]
	newMacro := macroFunction{
		fun: functionDef{
			name:       macroNameStr,
			eager:      true,
			quote:      false,
			parameters: macroParams,
			varargs:    varargs,
			bindings:   bindin,
		},
		body: macroBody,
	}
	bindin.bind[string(macroNameStr)] = newMacro
	return nily{}
}
