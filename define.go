package goyamp

import (
	"fmt"
	"log"
)

func map_define(defs mapy, bindings *env) yamly {
	log.Printf("map_define: %v\n", defs)
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

func define_builtin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("define_builtin:\n   %#v\n   %#v\n", tree, args)
	//    """
	//    Define one or more variables in the current scope.
	//    :return: None
	//    """
	assert_single_key(tree)
	todefine, aok := args.(mapy)
	if !aok {
		panic(fmt.Sprintf("define expects a map in %v got %T", tree, args))
	}
	name, nok := todefine[stringy("name")]
	value, vok := todefine[stringy("value")]
	log.Printf("define_builtin: nok vok %v %v\n", nok, vok)
	if !nok && !vok {
		return map_define(todefine, bindings)
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
	log.Printf("\ndefine_builtin:\n new value for :  %#v\n value:  %#v\n", namestr, newvalue)
	return nily{}
}

func undefine_builtin(tree mapy, args yamly, bindin *env) yamly {
	log.Printf("undefine_builtin:\n   %#v\n   %#v\n", tree, args)
	//    """
	//    Remove binding in the current environment only.
	//    :return: None
	//    """
	//  TODO  assert_single_key(tree)
	if variable, ok := args.(stringy); ok {
		delete(bindin.bind, string(variable))
		return nily{}
	}
	panic(fmt.Sprintf("Syntax error was expecting string in %v got %v", tree, args))
}

func defmacro_builtin(tree mapy, args yamly, bindin *env) yamly {
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
	macro_name, ok := argst[stringy("name")]
	if !ok {
		panic(fmt.Sprintf("missing names in %v", tree))
	}
	macro_name_str, ok := macro_name.(stringy)
	if !ok {
		panic(fmt.Sprintf("in macro %v is not a string", macro_name))
	}

	// Collate arguments
	var macro_params = seqy{} // default is list if no args
	varargs := false
	macro_p, ok := argst[stringy("args")]
	if ok {
		switch theparams := macro_p.(type) {
		case nily:
		case stringy: //varargs
			macro_params = append(macro_params, theparams)
			varargs = true
		case seqy:
			for _, k := range theparams {
				if param_name, ok := k.(stringy); ok {
					macro_params = append(macro_params, param_name)
				} else {
					panic(fmt.Sprintf("in defmacro args '%#v' is not a string", k))
				}
			}
		default:
			panic(fmt.Sprintf("in defmacro args '%#v' is not a string", macro_p))
		}
	}
	macro_body, ok := argst[stringy("value")]
	new_macro := macroFunction{
		fun: functionDef{
			name:       macro_name_str,
			eager:      true,
			quote:      false,
			parameters: macro_params,
			varargs:    varargs,
			bindings:   bindin,
		},
		body: macro_body,
	}
	bindin.bind[string(macro_name_str)] = new_macro
	return nily{}
}
