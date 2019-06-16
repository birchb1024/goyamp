package goyamp

import (
	"fmt"
	"log"
	"strconv"
)

func subvar_lookup(original string, vars_list []string, major_variable yamly, bindings *env) yamly {
	log.Printf("subvar_lookup: %v %v %#v %v\n", original, vars_list, major_variable, bindings)
	//    """
	//    Parse and expand a 'dot notation' variable string. Recursively walk the tree of the main variable value,
	//    as given by the subvariable list. Return the last value if possible.
	//
	//    :param original: The dot notation string - ie. 'b.1' - used for debug
	//    :param vars_list: a list of 'sub' variables - ie ['1']
	//    :param major_variable: the value of the major variable - ie. value of 'b' => ['x', 'y']
	//    :param bindings: the current environment
	//    :return: Example - Given 'b.1', ['b', '1' ] , {'b': ['x', 'y']} => returns 'y'
	//    """
	if len(vars_list) == 0 {
		panic(fmt.Sprintf("Subvariable not found in %v %v", original, major_variable))
	}
	if major_variable == nil {
		panic(fmt.Sprintf("Subvariable %v not found in %v and %v", vars_list, original, major_variable))
	}
	// If the subvar is a variable binding, use it
	var first yamly
	ftv, ok := bindings.lookup(stringy(vars_list[0]))
	log.Printf("subvar_lookup ftv ok: %v %v", ftv, ok)
	if ok {
		first = ftv
	} else {
		first = stringy(vars_list[0])
	}
	firststr, sok := first.(stringy)
	firstint, iok := first.(inty)
	if !sok && !iok {
		panic(fmt.Sprintf("Subvariable value %v for not string or int in %v", first, original))
	}
	// What kind of variable data do we have?
	switch major_variable_typed := major_variable.(type) {
	case mapy:
		// A map of values
		result, ok := major_variable_typed[firststr]
		if !ok {
			panic(fmt.Sprintf("Subvariable '%v' not found in %v and %v", first, original, major_variable_typed))
		}
		if len(vars_list) == 1 {
			return result
		} else {
			return subvar_lookup(original, vars_list[1:], result, bindings)
		}

	case seqy:
		// A Sequence needing an integer index
		var index int
		if iok {
			index = int(firstint)
		} else if sok {
			ind, err := strconv.Atoi(string(firststr))
			if err != nil {
				panic(fmt.Sprintf("Subvariable List index not numeric: '%v' for %v %v", first, original, major_variable))
			}
			index = ind
		} else {
			panic(fmt.Sprintf("Subvariable List index not numeric: '%v' for %v %v", first, original, major_variable))
		}

		if len(major_variable_typed) <= index || index < 0 {
			panic(fmt.Sprintf("Subvariable List index out of bounds: %v for %v %v", index, original, major_variable_typed))
		}
		if len(vars_list) == 1 { // Last one
			return major_variable_typed[index]
		} else {
			return subvar_lookup(original, vars_list[1:], major_variable_typed[index], bindings)
		}
	default:
		panic(fmt.Sprintf("Subvariable data not indexable %#v in %v", major_variable, original))
	}
}
