package goyamp

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var Version string

//
// 
type yamly interface {
	expand(binding *env) yamly
}

//
// Type nily stands instead of nil
// to avoid nil.expand() 
type nily struct{}
type booly bool
type inty int
type stringy string
type float64y float64
type mapy map[yamly]yamly
type seqy []yamly
type unknowny struct {
	x interface{}
}

func (x nily) expand(binding *env) yamly { return x }

func (x inty) expand(binding *env) yamly { return x }

func (x float64y) expand(binding *env) yamly { return x }

func (x booly) expand(binding *env) yamly { return x }

func (x unknowny) expand(binding *env) yamly { return x }

func (x unknowny) String() string { return fmt.Sprintf("unknown: %T %#v", x.x) }

func (x nily) String() string { return "null" }


// - Engine internals...
type env struct {
	bind   map[string]yamly
	parent *env
	engine *Expander
}

type Expander struct {
	globals *env
	output  io.Writer
}

func (e env) String() string { return fmt.Sprintf("an environment...") }

type builtin func(mapy, yamly, *env) yamly

type functionDef struct {
	name       stringy
	eager      bool
	quote      bool
	parameters seqy
	varargs    bool
	bindings   *env // Closure
}

type macroFunction struct {
	fun  functionDef
	body yamly
}

func (x macroFunction) expand(binding *env) yamly { return x }

type compiledFunction struct {
	fun      functionDef
	compiled builtin
}

func (x compiledFunction) expand(binding *env) yamly { return x }

type runnable interface {
	isEager() bool
	isQuote() bool

	apply(mapy, yamly, *env) yamly
}

func (s seqy) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s seqy) Less(i, j int) bool {
	si := fmt.Sprintf("%v", s[i]) // TODO remove use of fmt
	sj := fmt.Sprintf("%v", s[j])
	return si < sj
}

// Swap swaps the elements with indexes i and j.
func (s seqy) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

//
// Stringer for maps, print in order for reliable testing
//
func (m mapy) String() string {
	// store the keys in slice in sorted order
	keystr := []string{}
	keystr2key := map[string]yamly{}
	for k := range m {
		key_as_string := fmt.Sprintf("%v", k) // TODO
		keystr = append(keystr, key_as_string)
		keystr2key[key_as_string] = k
	}
	sort.Strings(keystr)

	var result string = "{"
	var counter int = 0
	for _, ks := range keystr {
		k := keystr2key[ks]
		v := m[k]
		counter += 1
		result = fmt.Sprintf("%v %v : %v ", result, k, v)
		if !(counter == len(m)) {
			result = fmt.Sprintf("%v,", result)
		}
	}
	result = fmt.Sprintf("%v }", result)
	return result
}

func (s seqy) String() string {
	var result string = "["
	for i, v := range s {
		result = fmt.Sprintf("%v %v", result, v)
		if i != len(s)-1 {
			result = fmt.Sprintf("%v,", result)
		}
	}
	result = fmt.Sprintf("%v ]", result)
	return result
}

//
// Convert something from the YAML parser into known
// types controlled in the goyamp packages.
//
func classify(x interface{}) yamly {
	switch x := x.(type) {
	case nil:
		return nily{}
	case int:
		return inty(x)
	case float64:
		return float64y(x)
	case bool:
		return booly(x)
	case string:
		return stringy(x)
	case map[interface{}]interface{}:
		result := mapy{}
		for k, v := range x {
			result[classify(k)] = classify(v)
		}
		return result
	case map[string]interface{}:
		result := mapy{}
		for k, v := range x {
			result[classify(k)] = classify(v)
		}
		return result
	case []interface{}:
		result := seqy{}
		for _, v := range x {
			result = append(result, classify(v))
		}
		return result
	case []string:
		result := seqy{}
		for _, v := range x {
			result = append(result, classify(v))
		}
		return result
	default:
		log.Printf("not classified %#v", x)
		return unknowny{x: x}
	}
}

//
//
func declassify(x yamly) interface{} {
	switch x := x.(type) {
	case nily:
		return nil
	case inty:
		return int(x)
	case float64y:
		return float64(x)
	case booly:
		return bool(x)
	case stringy:
		return string(x)
	case mapy:
		result := map[interface{}]interface{}{}
		for k, v := range x {
			result[declassify(k)] = declassify(v)
		}
		return result
	case seqy:
		result := []interface{}{}
		for _, v := range x {
			result = append(result, declassify(v))
		}
		return result
	default:
		return x
	}
}

func (r macroFunction) isEager() bool {
	log.Printf("isEager: %v", r)
	return r.fun.eager
}
func (r macroFunction) isQuote() bool {
	log.Printf("isQuote: %v", r)
	return r.fun.quote
}
func (r compiledFunction) isEager() bool {
	log.Printf("isEager: %v", r)
	return r.fun.eager
}
func (r compiledFunction) isQuote() bool {
	log.Printf("isQuote: %v", r)
	return r.fun.quote
}
func (r macroFunction) String() string {
	return fmt.Sprintf("macro: name: %v, eager: %v", r.fun.name, r.fun.eager)
}
func (r compiledFunction) String() string {
	return fmt.Sprintf("builtin: name: %v, eager: %v", r.fun.name, r.fun.eager)
}

var interpolate_regex *regexp.Regexp

func init() {
	interpolate_regex = regexp.MustCompile(`{{[^{]*}}`)
	log.SetFlags(log.Lshortfile)
}

func interpolate(tree yamly, bindings *env) yamly {

	//    """
	//    Parse a string which may contain embedded variables denoted by curlies {{ }}.
	//    When these are found expand the variables and return the expanded string.
	//    If the variables are called up but not defined throw an error.
	//    :param astring:
	//    :param bindings:
	//    :return: astring with added values
	//    """
	//
	s, ok := tree.(stringy)
	if !ok {
		log.Printf("interpolate not string %#v", tree)
		return tree
	}
	astring := string(s)
	log.Printf("interpolate: astring %#v", astring)
	result := interpolate_regex.ReplaceAllStringFunc(astring,
		func(tok string) string {
			variable_name := strings.TrimSpace(tok[2 : len(tok)-2])
			log.Printf("interpolate: variablename %#v", variable_name)
			value := expand_str(variable_name, bindings)
			log.Printf("interpolate: value, ok %#v %#v", value, ok)
			if !ok {
				panic(fmt.Sprintf("Undefined interpolation variable %v in %v", variable_name, astring))
			}
			return fmt.Sprintf("%v", value) // OK for scalars todo what about collections?
		})
	return stringy(result)
}

func (env *env) lookup(any yamly) (yamly, bool) {
	log.Printf("lookup: %#v\n", any)
	//    """
	//    Search an environment stack for a binding of key to a value,
	//    following __parent__ links to higher environment.
	//    :param env: Start seaching from this env
	//    :param key: variable name to look for.
	//    :return: value, ok - if key is found ok is True and value has the value, otherwise ok is False and value is undefined.
	//    """
	keyy, ok := any.(stringy)
	if !ok {
		return nil, false
	}
	key := string(keyy)
	for {
		val, ok := env.bind[key]
		if ok {
			return val, true
		}
		if env.parent != nil {
			env = env.parent
			continue
		} else {
			return nil, false
		}
	}
}

func (run compiledFunction) apply(seen_tree mapy, args yamly, dynamic_bindings *env) yamly {
	return run.compiled(seen_tree, args, dynamic_bindings)
}

func (run macroFunction) apply(seen_tree mapy, anyargs yamly, dynamic_bindings *env) yamly {
	log.Printf("apply:\n   %#v\n   %#v\n   %#v\n   %#v\n", seen_tree, anyargs, run)
	//        """
	//        Given a map of arguments, create a new local environment for this macro expansion, bind the args to the new
	//        enviroment, then expand the captured body and return the result. If the captured parameters variable is a string, it is
	//        used for variable arguments which are all bound to it.
	//        :param seen_tree: Tree as parsed
	//        :param args:
	//        :param dynamic_bindings: bindings for builtins
	//        :return:
	//        """
	fun := run.fun
	if len(seen_tree) != 1 { // Macros always have a single key
		panic(fmt.Sprintf("ERROR: too many keys in macro call '%v'", seen_tree))
	}
	args, ok := anyargs.(mapy)
	if args != nil && !ok {
		panic(fmt.Sprintf("Expecting argument map for %v, got: %#v", fun.name, args))
	}
	if fun.parameters == nil {
		panic(fmt.Sprintf("parameters are ni!!!"))
	}
	if len(fun.parameters) == 0 && args != nil {
		panic(fmt.Sprintf("Too many args for %v: %v", fun.name, args))
	}
	// Check all params are present
	if !fun.varargs && len(args) > 0 {
		missing := seqy{}
		for _, k := range fun.parameters {
			_, ok := args[k]
			if !ok {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			panic(fmt.Sprintf("Argument mismatch in %v expected %v got %v", fun.name, fun.parameters, args))
		}
		if len(missing) == 0 && len(args) > len(fun.parameters) {
			panic(fmt.Sprintf("To many arguments in %v expected %v got %v", fun.name, fun.parameters, args))
		}
	}
	// Now create env for macro call
	macro_env := env{
		engine: dynamic_bindings.engine,
		parent: fun.bindings,
		bind:   map[string]yamly{"__SOURCE__": seen_tree},
	}
	if fun.varargs {
		log.Printf("varags: %v %v\n", args, fun.parameters)
		varg := fun.parameters[0].(stringy)
		macro_env.bind[string(varg)] = anyargs
	} else {
		for k, v := range args {
			argname, ok := k.(stringy)
			if !ok {
				panic(fmt.Sprintf("arg name %v must be string in %v", k, seen_tree))
			}
			macro_env.bind[string(argname)] = v
		}
	}
	return run.body.expand(&macro_env)

}


func expand_str(var_name string, bindings *env) yamly {
	//    """
	//    Given a simple string variable get its value from the binding, it has dot notation look in the
	//    variable value for the selection.
	//    :param tree: - the variable name in a simple case, or the dotnotation variable.
	//    :param bindings: - current environment
	//    :return:
	//    """
	variable_name := stringy(var_name)
	value, ok := bindings.lookup(variable_name)
	if ok {
		return value // a simple variable like 'host' or a variable like 'a.c.e' matches first
	}
	//  simple, look for subvariables.
	subvar := strings.Split(var_name, ".")
	if len(subvar) > 1 {
		// It's a dot notation variable like 'host.name'
		topvalue, ok := bindings.lookup(stringy(subvar[0]))
		if !ok {
			return variable_name // No variable found
		}
		return subvar_lookup(var_name, subvar[1:], topvalue, bindings)
	} else {
		return variable_name
	}
}

func assert_single_key(tree yamly) {
	//    """
	//    Raise an exception if there are not a single key in map in tree.
	//    :return: None
	//    """
	amap, ok := tree.(mapy)
	if !ok {
		panic(fmt.Sprintf("Syntax error: aruments not a map."))
	}
	if len(amap) != 1 {
		panic(fmt.Sprintf("Syntax error too many keys in %v", amap))
	}
}

// TODO def validate_params(tree, tree_proto, args, args_proto):
//    """
//    Given a protype for a form and arguments, raise execptions if they dont match.
//    Checks:
//        - the number of keys in the tree,
//        - the type of the args
//        - the number of agrs if a list
//
//    e.g. validate_params({'a': None}, {'a': None}, [1], [1]) is OK
//    :return: None
//    """
//    if len(tree.keys()) != len(tree_proto):
//            raise(interface{}pException('Syntax error incorrect number of keys in {}'.format(tree)))
//    if type(args) != type(args_proto):
//            raise(interface{}pException('Syntax error incorrect argument type. Expected {} in {}'.format(type(args_proto), tree)))
//    if type(args) in [list, dict]:         # Is it someinterface{} with a length?
//        if len(args) < len(args_proto):
//                raise(interface{}pException('Syntax error too few arguments. Expected {} in {}'.format(len(args_proto), tree)))
//
//TODO def validate_keys(specification, amap):
//    """
//    Raise an exception if the keys in the specification are not present in the args, or if there are
//    additional keys not in the spec. Optional keys are wrapped in a tuple.
//    Example:
//       ['for', 'in', ('step')]
//    """
//    extras = set(amap.keys())
//    for key in specification:
//      if type(key) == str:
//        if not key in amap:
//           raise(interface{}pException('Syntax error missing argument {} in {}'.format(key, amap)))
//        extras.discard(key)
//      elif type(key) == tuple:
//         optional = key[0]
//         if type(optional) != str:
//            raise(interface{}pException('Invalid spec {}'.format(specification)))
//         extras.discard(optional)
//      else:
//          raise(interface{}pException('Invalid {} spec {}'.format(type(key), specification)))
//    if len(extras) > 0:
//        raise(interface{}pException('Unexpected keys {} in {}'.format(extras, amap)))
//

func any2int(item yamly, errorMessage string) int {
	switch item := item.(type) {
	case inty:
		return int(item)
	case stringy:
		ind, err := strconv.Atoi(string(item))
		if err != nil {
			panic(fmt.Sprintf(errorMessage, item))
		}
		return ind
	default:
		panic(fmt.Sprintf(errorMessage, item))
	}
}

func lookup_caret(key yamly, bindings *env) yamly {
	log.Printf("lookup_caret %#v\n", key)
	keysty, ok := key.(stringy)
	if !ok {
		return key
	}
	keyst := string(keysty)
	if !strings.HasPrefix(keyst, "^") {
		return key
	} else {
		variable_name := keyst[1:]
		if value, ok := bindings.lookup(stringy(variable_name)); ok {
			return value
		}
		panic(fmt.Sprintf("caret variable %v not defined in %v", variable_name, key))
	}
}

func is_function(tree mapy, bindings *env) (bool, runnable, yamly) {
	log.Printf("is_function %v\n", tree)

	// Return function tuple and rhs if this is a function call, else False

	lookup_function := func(k yamly) (runnable, bool) {
		// Return the function def from its binding, or not
		log.Printf("lookup_function %#v\n", k)

		function_name := lookup_caret(k, bindings)
		log.Printf("lookup_caret => %v\n", function_name)
		value, ok := bindings.lookup(function_name)
		log.Printf("lookup_function. value, ok  = %v, %v\n", value, ok)
		if !ok {
			return nil, false
		}
		if result, ok := value.(runnable); ok {
			log.Printf("lookup_function. result, ok  = %v, %v\n", result, ok)
			return result, true
		}
		return nil, false

	}
	var k yamly = nil
	if len(tree) == 1 {
		for key := range tree {
			k = key // executes once only
		}
	} else if _, ok := tree[stringy("if")]; ok { // Special case :-(
		k = stringy("if")
	}
	log.Printf("is_function k = %v\n", k)

	fun, ok := lookup_function(k)
	log.Printf("is_function fun = %#v %#v\n", fun, ok)
	if ok {
		log.Printf("is_function fun ok => %v\n", fun)
		return true, fun, tree[k]
	}
	log.Printf("is_function fun not ok => %v\n", fun)
	// At this point we have len(keys()) > 1 and its not an "if"
	// so we cannot have a function under any key...
	for k, _ := range tree {
		if _, ok := lookup_function(k); ok {
			panic(fmt.Sprintf("ERROR: too many keys in macro %v", tree))
		}
	}
	return false, nil, nil
}

func (tree seqy) expand(bindings *env) yamly {
	newlist := seqy{}
	for _, item := range tree {
		expanded := item.expand(bindings)
		if _, nok := expanded.(nily); !nok {
			newlist = append(newlist, expanded)
		}
	}
	return newlist
}

func (tree stringy) expand(bindings *env) yamly {
	result := expand_str(string(tree), bindings)
	if result == tree {
		return interpolate(tree, bindings)
	}
	if result_str, ok := result.(stringy); ok {
		return interpolate(result_str.expand(bindings), bindings)
	} else {
		return result.expand(bindings)
	}
}

func (tree_typed mapy) expand(bindings *env) yamly {

	newdict := mapy{}

	// Lookahead for functions. Some have Lazy maps we dont want to expand yet...
	if ok, function, rhs := is_function(tree_typed, bindings); ok {
		log.Printf("is_function returned => :\n    %v\n    %#v\n    %v\n", ok, function, rhs)
		var actual_args, expanded_result yamly

		if function.isEager() {
			actual_args = rhs.expand(bindings)
		} else { // lazy, quote
			actual_args = rhs
		}
		expanded_result = function.apply(tree_typed, actual_args, bindings)
		if function.isQuote() {
			return expanded_result
		} else {
			//			fmt.Printf("expanded_result = %#v bindings = %#v\n", expanded_result, bindings)
			return expanded_result.expand(bindings)
		}

	}
	// Just a normal map - not a function
	for k, v := range tree_typed {

		if kstr, ok := k.(stringy); ok {
			if strings.HasPrefix(string(kstr), "^") {
				variable_name := kstr[1:]
				value, ok := bindings.lookup(variable_name)
				if !ok {
					panic(fmt.Sprintf("ERROR: Variable %v not defined in %v", variable_name, tree_typed))
				}
				expanded := v.expand(bindings)
				if !reflect.ValueOf(value).Type().Comparable() {
					panic(fmt.Sprintf("Unable to use %v as map key, '%v' is not comparable. in %v", variable_name, value, tree_typed))
				}
				newdict[value] = expanded
				continue
			}
		}
		interp_k := interpolate(k, bindings)
		if interp_k != k {
			// string containing {{ }} - only these keys are expanded
			if _, ok := newdict[interp_k]; ok {
				panic(fmt.Sprintf("ERROR: duplicate map key '%v' in %v", interp_k, tree_typed))
			}
			newdict[interp_k] = v.expand(bindings)
			continue
		} else {
			log.Printf("not interpolating key %#v", k)
		}
		if _, ok := newdict[k]; ok {
			panic(fmt.Sprintf("ERROR: duplicate map key '%v' in %v", interp_k, tree_typed))
		}
		newdict[k] = v.expand(bindings)
	}
	return newdict
}

func expand_file(filename string, bindings *env) error {
	log.Printf("expand_file: filename %v\n", filename)

	var current_dir, path, prior string
	var err error
	current_path, ok := bindings.lookup(stringy("__PATH__")) // Remember prior file

	if !ok {
		log.Printf("expand_file: __PATH__ => not bound\n")
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		current_dir, err = filepath.Abs(pwd)
		if err != nil {
			log.Panic(err)
		}
	} else {
		prior = string(current_path.(stringy))
		log.Printf("expand_file: __PATH__ => %v\n", prior)
		current_dir = filepath.Dir(prior)
	}
	log.Printf("expand_file: current_dir => %v\n", current_dir)

	if strings.HasPrefix(filename, "/") || filename == "-" {
		path = filename
	} else {
		path, err = filepath.Abs(filepath.Join(current_dir, filename)) // resolve relative paths
		if err != nil {
			return err
		}
	}
	bindings.bind["__PATH__"] = stringy(path) // New file now
	bindings.bind["__FILE__"] = stringy(filepath.Base(path))
	log.Printf("expand_file: new __FILE__ => %v\n", filepath.Base(path))
	log.Printf("expand_file: new __PATH__ => %v\n", path)
	input, err := os.Open(path)
	if err != nil {
		log.Printf("open => %v %v %v\n", path, input, err)
		return err
	}
	err = expand_stream(input, filename, bindings)
	if err != nil {
		return err
	}
	bindings.bind["__PATH__"] = current_path // restore prior file
	bindings.bind["__FILE__"] = stringy(filepath.Base(prior))
	return nil
}

func expand_stream(input io.Reader, filename string, bindings *env) (err error) {
	decoder := yaml.NewDecoder(input)
	for {
		var doc interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		ya := classify(doc)
		expanded := ya.expand(bindings)
		if _, nilok := expanded.(nily); nilok {
			log.Printf("expand_steam: doc is null\n")
			continue
		}
		if array, ok := expanded.(seqy); ok {
			if len(array) == 0 {
				continue
			}
		}
		enc := yaml.NewEncoder(bindings.engine.output)
		enc.SetIndent(2)
		fmt.Fprintln(bindings.engine.output, "---")
		err = enc.Encode(declassify(expanded))
		if err != nil {
			return err
		}
	}
}

var debugFlag bool = false
