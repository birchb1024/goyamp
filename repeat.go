package goyamp

import (
	"fmt"
	"log"
	"sort"
)

func repeatMap(tree yamly, forvariablename stringy, keyvar yamly, rangeList seqy, body yamly, bindings *env) yamly {
	log.Printf("repeatMap: %v %v %v %v", tree, forvariablename, rangeList, body)
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
	//def expandRepeatList(tree, statement, bindings):
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

func repeatBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("repeatBuiltin: %v %v %v\n", tree, args, bindings)
	//    """
	//    Expand a repeat macro, this function selects the appropriate expander for lists and maps.
	//    If the repeat has the 'key' key, then execute as for maps, else lists.
	//    :param tree: The repeat form such as {repeat: {for: X, in: [1,2], key: 'Foo {{X}}', body: [stuff, X]}
	//    :param bindings:
	//    :return: The Expanse
	//    """
	treemap, ok := args.(mapy)
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map, got %v\n", tree))
	}
	assertKeys(map[string]bool{"for": true, "in": true, "body": true, "key": false}, treemap)

	forClause := treemap[stringy("for")]
	forVariable, ok := forClause.(stringy)
	if !ok {
		panic(fmt.Sprintf("Syntax error repeat expects a map with string 'for', got %v", tree))
	}

	inClause, ok := treemap[stringy("in")]
	rangein := inClause.expand(bindings)
	rangeinList := seqy{}
	switch item := rangein.(type) {
	case seqy:
		rangeinList = item
	case mapy:
		// Convert to sorted list of keys
		rangeinList = make([]yamly, 0, len(item))
		for k := range item {
			rangeinList = append(rangeinList, k)
		}
		sort.Sort(rangeinList)
	default:
		panic(fmt.Sprintf("Syntax error in repeat 'in' is not a map or sequence, got %#v in %v", rangein, tree))
	}

	body, ok := treemap[stringy("body")]
	loopEnv := env{
		engine: bindings.engine,
		parent: bindings,
		bind:   map[string]yamly{},
	}
	if keyvar, ok := treemap[stringy("key")]; ok {
		return repeatMap(tree, forVariable, keyvar, rangeinList, body, &loopEnv)
	}
	return repeatSequence(tree, forVariable, rangeinList, body, &loopEnv)

}
