package goyamp

import (
	"fmt"
	"log"
	"strconv"
)

func subvarLookup(original string, varsList []string, majorVariable yamly, bindings *env) yamly {
	log.Printf("subvarLookup: %v %v %#v %v\n", original, varsList, majorVariable, bindings)
	//    """
	//    Parse and expand a 'dot notation' variable string. Recursively walk the tree of the main variable value,
	//    as given by the subvariable list. Return the last value if possible.
	//
	//    :param original: The dot notation string - ie. 'b.1' - used for debug
	//    :param varsList: a list of 'sub' variables - ie ['1']
	//    :param majorVariable: the value of the major variable - ie. value of 'b' => ['x', 'y']
	//    :param bindings: the current environment
	//    :return: Example - Given 'b.1', ['b', '1' ] , {'b': ['x', 'y']} => returns 'y'
	//    """
	if len(varsList) == 0 {
		panic(fmt.Sprintf("Subvariable not found in %v %v", original, majorVariable))
	}
	if majorVariable == nil {
		panic(fmt.Sprintf("Subvariable %v not found in %v and %v", varsList, original, majorVariable))
	}
	// If the subvar is a variable binding, use it
	var first yamly
	ftv, ok := bindings.lookup(stringy(varsList[0]))
	log.Printf("subvarLookup ftv ok: %v %v", ftv, ok)
	if ok {
		first = ftv
	} else {
		first = stringy(varsList[0])
	}
	firststr, sok := first.(stringy)
	firstint, iok := first.(inty)
	if !sok && !iok {
		panic(fmt.Sprintf("Subvariable value %v for not string or int in %v", first, original))
	}
	// What kind of variable data do we have?
	switch majorVariableTyped := majorVariable.(type) {
	case mapy:
		// A map of values
		result, ok := majorVariableTyped[firststr]
		if !ok {
			panic(fmt.Sprintf("Subvariable '%v' not found in %v and %v", first, original, majorVariableTyped))
		}
		if len(varsList) == 1 {
			return result
		}
		return subvarLookup(original, varsList[1:], result, bindings)

	case seqy:
		// A Sequence needing an integer index
		var index int
		if iok {
			index = int(firstint)
		} else if sok {
			ind, err := strconv.Atoi(string(firststr))
			if err != nil {
				panic(fmt.Sprintf("Subvariable List index not numeric: '%v' for %v %v", first, original, majorVariable))
			}
			index = ind
		} else {
			panic(fmt.Sprintf("Subvariable List index not numeric: '%v' for %v %v", first, original, majorVariable))
		}

		if len(majorVariableTyped) <= index || index < 0 {
			panic(fmt.Sprintf("Subvariable List index out of bounds: %v for %v %v", index, original, majorVariableTyped))
		}
		if len(varsList) == 1 { // Last one
			return majorVariableTyped[index]
		}
		return subvarLookup(original, varsList[1:], majorVariableTyped[index], bindings)

	default:
		panic(fmt.Sprintf("Subvariable data not indexable %#v in %v", majorVariable, original))
	}
}
