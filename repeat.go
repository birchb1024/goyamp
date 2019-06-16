package goyamp

import (
	"fmt"
	"log"
)

func repeat_map(tree yamly, forvariablename stringy, keyvar yamly, rangeList seqy, body yamly, bindings *env) yamly {
	log.Printf("repeat_map: %v %v %v %v", tree, forvariablename, rangeList, body)
	//    """
	//    Expand a repeat loop and return a map, with a parameterized key. Create a local environment for the
	//    expansion, bind the for variable name to the iteration value each time round.
	//    :param tree: The repeat form such as {repeat: {for: X, in: [1,2], key: 'Foo {{X}}', body: [stuff, X]}
	//    :param bindings:
	//    :return: The Expanse
	//    """
	result := mapy{}
	for _, item := range rangeList {
		bindings.bind[string(forvariablename)] = item
		keyvalue := keyvar.expand(bindings)
		if _, ok := result[keyvalue]; ok {
			panic(fmt.Sprintf("ERROR: key '%v' duplication in %v", keyvalue, tree))
		}
		result[keyvalue] = body.expand(bindings)
	}
	return result
}

func repeatSequence(tree yamly, forvariablename stringy, rangeList seqy, body yamly, bindings *env) yamly {
	log.Printf("repeatSequence: %v %v %v %v %v\n", tree, forvariablename, rangeList, body, bindings)
	//def expand_repeat_list(tree, statement, bindings):
	//    """
	//    Expand a repeat loop and return a list one item each time. Create a local environment for the
	//    expansion, bind the for variable name to the iteration value each time round.
	//    :param tree: The repeat form such as {repeat: {for: X, in: [1,2], body: [stuff, X]}
	//    :param bindings:
	//    :return: The Expanse
	//    """
	result := seqy{}
	for _, item := range rangeList {
		bindings.bind[string(forvariablename)] = item
		result = append(result, body.expand(bindings))
	}
	return result
}

func repeat_builtin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("repeat_builtin: %v %v %v\n", tree, args, bindings)
	//    """
	//    Expand a repeat macro, this function selects the appropriate expander for lists and maps.
	//    If the repeat has the 'key' key, then execute as for maps, else lists.
	//    :param tree: The repeat form such as {repeat: {for: X, in: [1,2], key: 'Foo {{X}}', body: [stuff, X]}
	//    :param bindings:
	//    :return: The Expanse
	//    """
	//    TODO validate_keys(['for', 'in', 'body', ('key',)], args)
	treemap, ok := args.(mapy)
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map, got %v\n", tree))
	}
	// TODO DRY...
	for_clause, ok := treemap[stringy("for")]
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map with 'for', got %v\n", tree))
	}
	for_variable, ok := for_clause.(stringy)
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map with string 'for', got %v", tree))
	}

	in_clause, ok := treemap[stringy("in")]
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map with 'in', got %v", tree))
	}
	rangein := in_clause.expand(bindings)
	rangein_list, ok := rangein.(seqy)
	if !ok {
		panic(fmt.Sprintf("Syntax error in repeat 'in' is not a sequnce, got %#v in %v", rangein, tree))
	}

	body, ok := treemap[stringy("body")]
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map with 'body', got %v", tree))
	}
	loop_env := env{
		engine: bindings.engine,
		parent: bindings,
		bind:   map[string]yamly{},
	}
	if keyvar, ok := treemap[stringy("key")]; ok {
		return repeat_map(tree, for_variable, keyvar, rangein_list, body, &loop_env)
	}
	return repeatSequence(tree, for_variable, rangein_list, body, &loop_env)

}
