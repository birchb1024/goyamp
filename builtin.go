package goyamp

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sort"
	"strconv"
)

//    """
//    :return: True or False depending if args are the same.
//    """
func equalsBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("== %v %v\n", tree, args)
	argarray, ok := args.(seqy)
	if !ok {
		panic(fmt.Sprintf("== builtin expected sequence, got %v", args))
	}
	if len(argarray) < 2 {
		panic(fmt.Sprintf("== builtin expected more than two in sequence, got %v", args))
	}
	expected := argarray[0]
	for _, item := range argarray {
		if !reflect.DeepEqual(item, expected) {
			return booly(false)
		}
	}
	return booly(true)
}

//    """
//    :return: the sum of the arguments.
//    """
func plusBuiltin(tree mapy, args yamly, bindings *env) yamly {
	argarray, ok := args.(seqy)
	if !ok {
		panic(fmt.Sprintf("+ builtin exepected sequence, got %v", args))
	}
	if len(argarray) < 2 {
		panic(fmt.Sprintf("+ builtin exepected more than two in sequence, got %v", args))
	}
	var sum float64
	for _, item := range argarray {
		switch toadd := item.(type) {
		case stringy:
			ind, err := strconv.Atoi(string(toadd))
			if err != nil {
				panic(fmt.Sprintf("+ builtin cannot add with %v", toadd))
			}
			sum += float64(ind)
		case inty:
			sum += float64(int(inty(toadd)))
		case float64y:
			sum += float64(float64y(toadd))
		default:
			panic(fmt.Sprintf("+ builtin cannot add with %v", toadd))
		}
	}
	if math.Round(sum) == sum {
		return inty(int(sum))
	}
	return float64y(sum)
}

//    """
//    :return: a list from  statement[0] to statement[1]
//    """
func rangeBuiltin(tree mapy, args yamly, bindings *env) yamly {
	switch args := args.(type) {
	case seqy:
		if len(args) != 2 {
			panic(fmt.Sprintf("range builtin needs sequence of 2, got %v", args))
		}
		start := any2int(args[0], "range builtin needs integers, got %v")
		end := any2int(args[1], "range builtin needs integers, got %v")
		result := seqy{}
		if start < end {
			for i := start; i < end+1; i++ {
				result = append(result, inty(i))
			}
		} else {
			for i := start; i >= end; i-- {
				result = append(result, inty(i))
			}
		}
		if len(result) == 0 {
			panic(fmt.Sprintf("range empty range in %v ", tree))
		}
		return result
	case mapy:
		result := seqy{}
		for key := range args {
			result = append(result, key)
		}
		sort.Sort(result)
		return result
	default:
		panic(fmt.Sprintf("range builtin needs sequence or map, got %v", args))
	}
}

//    """
//    expand and flatten a list.
//    :param bindings:
//    :return:
//    """
func flattenList(any yamly, depth int) yamly {
	log.Printf("flattenList: %v", any)

	if depth == 0 {
		return any
	}
	switch listy := any.(type) {
	case seqy:
		result := seqy{}
		for _, item := range listy {
			switch item := item.(type) {
			case seqy:
				sub := flattenList(item, depth-1)
				if sublist, ok := sub.(seqy); ok {
					for _, v := range sublist {
						result = append(result, v)
					}
				} else {
					result = append(result, item)
				}
			default:
				result = append(result, item)
			}
		}
		return result
	default:
		return any
	}
}

func flattenBuiltin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See flattenList
	//    """
	return flattenList(args, math.MaxInt32)
}

func flatoneBuiltin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See flatList
	//    """
	return flattenList(args, 1)
}

func mergeBuiltin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See mergeMaps
	//    """
	//    """
	//    Expand and combine multiple maps into one map. Not recursive. Later maps overwrite earlier.
	//    :param mappy: list of maps to be merged.
	//    :param bindings:
	//    :return: new map with merged content
	//    """
	result := make(mapy)
	switch args := args.(type) {
	case seqy:
		for _, item := range args {
			switch item := item.(type) {
			case mapy:
				for k, v := range item {
					result[k] = v // later overwrites earlier
				}
			default:
				panic(fmt.Sprintf("Error: non-map passed to merge '%v' from %v", item, tree))
			}
		}
		return result
	default:
		return args
	}
}

func assertKeys(validKeys map[string]bool, tmap mapy) {
	for x := range tmap {
		x, ok := x.(stringy)
		if !ok {
			panic(fmt.Sprintf("'%v' is not a valid key in %v", x, tmap))
		}
		if _, ok := validKeys[string(x)]; !ok {
			panic(fmt.Sprintf("'%v' is not a valid key in %v", x, tmap))
		}
	}
	for k, mandatory := range validKeys {
		if mandatory {
			if _, ok := tmap[stringy(k)]; !ok {
				panic(fmt.Sprintf("missing key '%v' in %v", mandatory, tmap))
			}
		}
	}
}

func ifBuiltin(tmap mapy, args yamly, bindings *env) yamly {
	log.Printf("ifBuiltin %v\n", tmap)
	//    """
	//    Conditional expression
	//    :return: either the expansion of the 'then' or 'else' elements.
	//    """
	assertKeys( map[string]bool{"if": true, "then": false, "else": false} , tmap)
	thenClause, thok := tmap[stringy("then")]
	elseClause, elok := tmap[stringy("else")]
	if !thok && !elok {
		panic(fmt.Sprintf("Syntax error 'then' or 'else' missing in %v", tmap))
	}
	condition := tmap[stringy("if")].expand(bindings)
	var condBool bool
	switch condition := condition.(type) {
	case empty:
		condBool = false
	case nily:
		condBool = false
	case booly:
		condBool = bool(condition)
	default:
		panic(fmt.Sprintf("If condition not 'true', 'false' or 'null'. Got: '%v' in %v", condition, tmap))
	}
	log.Printf("ifBuiltin: condBool %v thok %v elok %v\n", condBool, thok, elok)
	if condBool && thok {
		expanded := thenClause.expand(bindings)
		return expanded.expand(bindings)
	} else if !condBool && elok {
		expanded := elseClause.expand(bindings)
		return expanded.expand(bindings)
	}
	return empty{}
}

func quoteBuiltin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    :return: the args without expansion
	//    """
	assertSingleKey(tree)
	return args
}

func addBuiltinsToEnv(env *env) {
	//    """
	//    Utility function to add all the builtins to an environment
	//    :env: environment to add to
	//    :return: The environment
	//    """
	addNewBuiltin := func(name stringy, fn builtin, eager bool, quote bool) {
		env.bind[string(name)] = compiledFunction{
			fun: functionDef{
				eager:      eager,
				quote:      quote,
				name:       name,
				parameters: seqy{stringy("varargs")},
				varargs:    true,
				bindings:   env,
			},
			compiled: fn,
		}
	}
	addNewBuiltin("define", defineBuiltin, false, false)
	addNewBuiltin("include", includeBuiltin, true, false)
	addNewBuiltin("defmacro", defmacroBuiltin, false, false)
	addNewBuiltin("==", equalsBuiltin, true, false)
	addNewBuiltin("+", plusBuiltin, true, false)
	addNewBuiltin("if", ifBuiltin, false, false)
	addNewBuiltin("range", rangeBuiltin, true, false)
	addNewBuiltin("repeat", repeatBuiltin, false, false)
	addNewBuiltin("quote", quoteBuiltin, false, true)
	addNewBuiltin("undefine", undefineBuiltin, false, false)
	addNewBuiltin("flatten", flattenBuiltin, true, false)
	addNewBuiltin("flatone", flatoneBuiltin, true, false)
	addNewBuiltin("merge", mergeBuiltin, true, false)
	addNewBuiltin("load", loadBuiltin, true, false)
	addNewBuiltin("execute", executeBuiltin, true, false)
	addNewBuiltin("panic", panicBuiltin, true, false)
}
