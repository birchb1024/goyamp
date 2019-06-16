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
//    validate_params(tree, {'': None}, args, [1, 2])
func equals_builtin(tree mapy, args yamly, bindings *env) yamly {
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
//    validate_params(tree, {'': None}, args, [1, 2])
func plus_builtin(tree mapy, args yamly, bindings *env) yamly {
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
	} else {
		return float64y(sum)
	}
}

//    """
//    :return: a list from  statement[0] to statement[1]
//    """
//    validate_params(tree, {'range': None}, statement, [1,2])
func range_builtin(tree mapy, args yamly, bindings *env) yamly {
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
		for key, _ := range args {
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
func flatten_list(any yamly, depth int) yamly {
	log.Printf("flatten_list: %v", any)

	if depth == 0 {
		return any
	}
	switch listy := any.(type) {
	case seqy:
		result := seqy{}
		for _, item := range listy {
			switch item := item.(type) {
			case seqy:
				sub := flatten_list(item, depth-1)
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

func flatten_builtin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See flatten_list
	//    """
	//  TODO  validate_params(tree, {'': None}, args, [])
	return flatten_list(args, math.MaxInt32)
}

func flatone_builtin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See flat_list
	//    """
	//   TODO validate_params(tree, {'': None}, args, [])
	return flatten_list(args, 1)
}

func merge_builtin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    See merge_maps
	//    """
	//    """
	//    Expand and combine multiple maps into one map. Not recursive. Later maps overwrite earlier.
	//    :param mappy: list of maps to be merged.
	//    :param bindings:
	//    :return: new map with merged content
	//    """
	//  TODO  validate_params(tree, {'': None}, args, [])
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

func if_builtin(tmap mapy, args yamly, bindings *env) yamly {
	log.Printf("if_builtin %v\n", tmap)
	//    """
	//    Conditional expression
	//    :return: either the expansion of the 'then' or 'else' elements.
	//    """
	then_clause, thok := tmap[stringy("then")]
	else_clause, elok := tmap[stringy("else")]
	if !thok && !elok {
		panic(fmt.Sprintf("Syntax error 'then' or 'else' missing in %v", tmap))
	}
	if len(tmap) > 3 {
		//   TODO extras = set(tree.keys()) - set(['if', 'then', 'else'])
		//   TODO  if extras:
		panic(fmt.Sprintf("Syntax error extra keys in %v", tmap))
	}
	condition := tmap[stringy("if")].expand(bindings)
	var cond_bool bool
	switch condition := condition.(type) {
	case booly:
		cond_bool = bool(condition)
	default:
		if _, nok := condition.(nily); nok {
			cond_bool = false
		} else {
			cond_bool = true
		}
	}
	log.Printf("if_builtin: cond_bool %v thok %v elok %v\n", cond_bool, thok, elok)
	// TODO       raise(interface{}pException('If condition not "true", "false" or "null". Got: "{}" in {}'.format(condition, tree)))
	if cond_bool && thok {
		expanded := then_clause.expand(bindings)
		return expanded.expand(bindings)
	} else if !cond_bool && elok {
		expanded := else_clause.expand(bindings)
		return expanded.expand(bindings)
	}
	return nily{}
}

func quote_builtin(tree mapy, args yamly, bindings *env) yamly {
	//    """
	//    :return: the args without expansion
	//    """
	// TODO    assert_single_key(tree)
	return args
}

func add_builtins_to_env(env *env) {
	//    """
	//    Utility function to add all the builtins to an environment
	//    :env: environment to add to
	//    :return: The environment
	//    """
	add_new_builtin := func(name stringy, fn builtin, eager bool, quote bool) {
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
	add_new_builtin("define", define_builtin, false, false)
	add_new_builtin("include", include_builtin, true, false)
	add_new_builtin("defmacro", defmacro_builtin, false, false)
	add_new_builtin("==", equals_builtin, true, false)
	add_new_builtin("+", plus_builtin, true, false)
	add_new_builtin("if", if_builtin, false, false)
	add_new_builtin("range", range_builtin, true, false)
	add_new_builtin("repeat", repeat_builtin, false, false)
	add_new_builtin("quote", quote_builtin, false, true)
	add_new_builtin("undefine", undefine_builtin, false, false)
	add_new_builtin("flatten", flatten_builtin, true, false)
	add_new_builtin("flatone", flatone_builtin, true, false)
	add_new_builtin("merge", merge_builtin, true, false)
	add_new_builtin("load", load_builtin, true, false)
	add_new_builtin("execute", execute_builtin, true, false)
	add_new_builtin("panic", panic_builtin, true, false)
}
