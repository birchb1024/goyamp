package goyamp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v3"
)

//
// Version provides the Git version tag used in the build of the binary
var Version string
var documentCount int

//
//
type yamly interface {
	expand(binding *env) yamly
	declassify(params ...Syntax) interface{}
	String() string
	gopherluaify(L *lua.LState) lua.LValue
}

//
// Type nily stands instead of nil
// to avoid nil.expand()
type nily struct{}
type empty struct{}
type booly bool
type inty int
type stringy string
type float64y float64
type mapy map[yamly]yamly
type seqy []yamly
type unknowny struct {
	x interface{}
}

func (x nily) expand(*env) yamly { return x }
func (x nily) String() string    { return "null" }

func (e empty) expand(*env) yamly { return e }
func (e empty) String() string    { return "goyamp.EMPTY" }

func (x inty) expand(*env) yamly { return x }
func (x inty) String() string    { return strconv.Itoa(int(x)) }

func (x float64y) expand(*env) yamly { return x }
func (x float64y) String() string    { return strconv.FormatFloat(float64(x), 'G', -1, 32) }

func (x booly) expand(*env) yamly { return x }
func (x booly) String() string {
	if x {
		return "true"
	}
	return "false"
}

func (x unknowny) expand(*env) yamly { return x }
func (x unknowny) String() string    { return fmt.Sprintf("unknown: %T %#v", x.x, x.x) }

func (x stringy) String() string { return string(x) }

// - Engine internals...
type env struct {
	bind   map[string]yamly
	parent *env
	engine *Expander
}

// Expander holds the state of the goyamp engine.
// It is not a singleton type, have as many as you want.
type Expander struct {
	globals   *env
	output    io.Writer
	outFormat Syntax
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

func (r macroFunction) expand(*env) yamly { return r }

type compiledFunction struct {
	fun      functionDef
	compiled builtin
}

func (r compiledFunction) expand(*env) yamly { return r }

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
	si := s[i].String()
	sj := s[j].String()
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
		keyAsString := k.String()
		keystr = append(keystr, keyAsString)
		keystr2key[keyAsString] = k
	}
	sort.Strings(keystr)

	result := "{"
	counter := 0
	for _, ks := range keystr {
		k := keystr2key[ks]
		v := m[k]
		counter++
		result = fmt.Sprintf("%v %v : %v ", result, k, v)
		if !(counter == len(m)) {
			result = fmt.Sprintf("%v,", result)
		}
	}
	result = fmt.Sprintf("%v }", result)
	return result
}

func (s seqy) String() string {
	result := "["
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
func (x nily) declassify(...Syntax) interface{}     { return nil }
func (e empty) declassify(...Syntax) interface{}    { return "goyamp.EMPTY" }
func (x inty) declassify(...Syntax) interface{}     { return int(x) }
func (x float64y) declassify(...Syntax) interface{} { return float64(x) }
func (x booly) declassify(...Syntax) interface{}    { return bool(x) }
func (x stringy) declassify(...Syntax) interface{}  { return string(x) }
func (m mapy) declassify(syntax ...Syntax) interface{} {
	if len(syntax) > 0 && syntax[0] == JSON {
		result := map[string]interface{}{}
		for k, v := range m {
			switch k := k.(type) {
			case nily:
				result["null"] = v.declassify(syntax...)
			default:
				result[fmt.Sprintf("%v", k.declassify(syntax...))] = v.declassify(syntax...)
			}
		}
		return result
	}
	result := map[interface{}]interface{}{}
	for k, v := range m {
		result[k.declassify()] = v.declassify(syntax...)
	}
	return result
}

func (s seqy) declassify(syntax ...Syntax) interface{} {
	result := []interface{}{}
	for _, v := range s {
		result = append(result, v.declassify(syntax...))
	}
	return result
}
func (r macroFunction) declassify(...Syntax) interface{}    { return r }
func (r compiledFunction) declassify(...Syntax) interface{} { return r }
func (x unknowny) declassify(...Syntax) interface{}         { return x.x }

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

var interpolateRegex *regexp.Regexp
var goyampExecutablePath string

func init() {
	interpolateRegex = regexp.MustCompile(`{{[^{]*}}`)
	log.SetFlags(log.Lshortfile)
	documentCount = 0
	// Locate current running goyamp process file
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	goyampExecutablePath = filepath.Dir(ex)
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
	result := interpolateRegex.ReplaceAllStringFunc(astring,
		func(tok string) string {
			variableName := strings.TrimSpace(tok[2 : len(tok)-2])
			log.Printf("interpolate: variablename %#v", variableName)
			// value, ok := bindings.lookup(stringy(variableName))
			value, ok := expandStr(variableName, bindings)
			if !ok {
				panic(fmt.Sprintf("'%v' is not a bound variable in '%v'", variableName, tree))
			}
			log.Printf("interpolate: value, ok %#v %#v", value, ok)
			return value.String()
		})
	return stringy(result)
}

func (e *env) lookup(any yamly) (yamly, bool) {
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
		val, ok := e.bind[key]
		if ok {
			return val, true
		}
		if e.parent != nil {
			e = e.parent
			continue
		}
		return nil, false
	}
}

func (f functionDef) checkArgumentsOrDie(actuals yamly) {
	log.Printf("checkArgumentsOrDie:\n   %#v\n   %#v\n", f, actuals)
	if f.varargs {
		return
	}
	switch args := actuals.(type) {
	case nily:
		return
	case mapy:

		if f.parameters == nil {
			panic(fmt.Sprintf("function parameters are nil!!"))
		}
		// Check all params are present
		missing := seqy{}
		for _, k := range f.parameters {
			_, ok := args[k]
			if !ok {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			panic(fmt.Sprintf("Argument mismatch in %v expected %v got %v", f.name, f.parameters, args))
		}
		if len(args) > len(f.parameters) {
			panic(fmt.Sprintf("To many arguments in %v expected %v got %v", f.name, f.parameters, args))
		}
	default:
		panic(fmt.Sprintf("Expecting argument map for %v, got: %#v", f.name, actuals))
	}
}

func (r compiledFunction) apply(seenTree mapy, actuals yamly, dynamicBindings *env) yamly {
	return r.compiled(seenTree, actuals, dynamicBindings)
}

func (r macroFunction) apply(seenTree mapy, anyargs yamly, dynamicBindings *env) yamly {
	log.Printf("apply:\n   %#v\n   %#v\n   %#v\n", seenTree, anyargs, r)
	//        """
	//        Given a map of arguments, create a new local environment for this macro expansion, bind the args to the new
	//        enviroment, then expand the captured body and return the result. If the captured parameters variable is a string, it is
	//        used for variable arguments which are all bound to it.
	//        :param seenTree: Tree as parsed
	//        :param args:
	//        :param dynamicBindings: bindings for builtins
	//        :return:
	//        """
	if len(seenTree) != 1 { // Macros always have a single key
		panic(fmt.Sprintf("ERROR: too many keys in macro call '%v'", seenTree))
	}
	r.fun.checkArgumentsOrDie(anyargs)

	macroEnv := env{
		engine: dynamicBindings.engine,
		parent: r.fun.bindings,
		bind:   map[string]yamly{"__SOURCE__": seenTree},
	}
	if r.fun.varargs {
		log.Printf("varags: %v %v\n", anyargs, r.fun.parameters)
		varg := r.fun.parameters[0].(stringy)
		macroEnv.bind[string(varg)] = anyargs
	} else {
		switch args := anyargs.(type) {
		case mapy:
			for k, v := range args {
				argname, ok := k.(stringy)
				if !ok {
					panic(fmt.Sprintf("arg name %v must be string in %v", k, seenTree))
				}
				macroEnv.bind[string(argname)] = v
			}
		case nily:

		default:
			panic(fmt.Sprintf("Expecting argument map or null for %v, got: %#v", r.fun.name, anyargs))
		}
	}
	return r.body.expand(&macroEnv)

}

func expandStr(varName string, bindings *env) (yamly, bool) {
	//    """
	//    Given a simple string variable get its value from the binding, it has dot notation look in the
	//    variable value for the selection.
	//    :param tree: - the variable name in a simple case, or the dotnotation variable.
	//    :param bindings: - current environment
	//    :return:
	//    """
	variableName := stringy(varName)
	value, ok := bindings.lookup(variableName)
	if ok {
		return value, true // a simple variable like 'host' or a variable like 'a.c.e' matches first
	}
	//  simple, look for subvariables.
	subvar := strings.Split(varName, ".")
	if len(subvar) > 1 {
		// It's a dot notation variable like 'host.name'
		topvalue, ok := bindings.lookup(stringy(subvar[0]))
		if !ok {
			return nil, false // No variable found
		}
		return subvarLookup(varName, subvar[1:], topvalue, bindings), true
	}
	return nil, false
}

func assertSingleKey(tree yamly) {
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

func lookupCaret(key yamly, bindings *env) yamly {
	log.Printf("lookupCaret %#v\n", key)
	keysty, ok := key.(stringy)
	if !ok {
		return key
	}
	keyst := string(keysty)
	if !strings.HasPrefix(keyst, "^") {
		return key
	}
	variableName := keyst[1:]
	if value, ok := bindings.lookup(stringy(variableName)); ok {
		return value
	}
	panic(fmt.Sprintf("caret variable %v not defined in %v", variableName, key))
}

func isFunction(tree mapy, bindings *env) (bool, runnable, yamly) {
	log.Printf("isFunction %v\n", tree)

	// Return function tuple and rhs if this is a function call, else False

	lookupFunction := func(k yamly) (runnable, bool) {
		// Return the function def from its binding, or not
		log.Printf("lookupFunction %#v\n", k)

		functionName := lookupCaret(k, bindings)
		log.Printf("lookupCaret => %v\n", functionName)
		value, ok := bindings.lookup(functionName)
		log.Printf("lookupFunction. value, ok  = %v, %v\n", value, ok)
		if !ok {
			return nil, false
		}
		if result, ok := value.(runnable); ok {
			log.Printf("lookupFunction. result, ok  = %v, %v\n", result, ok)
			return result, true
		}
		return nil, false

	}
	var k yamly
	if len(tree) == 1 {
		for key := range tree {
			k = key // executes once only
		}
	} else if _, ok := tree[stringy("if")]; ok { // Special case :-(
		k = stringy("if")
	}
	log.Printf("isFunction k = %v\n", k)

	fun, ok := lookupFunction(k)
	log.Printf("isFunction fun = %#v %#v\n", fun, ok)
	if ok {
		log.Printf("isFunction fun ok => %v\n", fun)
		return true, fun, tree[k]
	}
	log.Printf("isFunction fun not ok => %v\n", fun)
	// At this point we have len(keys()) > 1 and its not an "if"
	// so we cannot have a function under any key...
	for k := range tree {
		if _, ok := lookupFunction(k); ok {
			panic(fmt.Sprintf("ERROR: too many keys in macro %v", tree))
		}
	}
	return false, nil, nil
}

func (s seqy) expand(bindings *env) yamly {
	newlist := seqy{}
	for _, item := range s {
		expanded := item.expand(bindings)
		if _, ok := expanded.(empty); !ok {
			newlist = append(newlist, expanded)
		}
	}
	return newlist
}

func (x stringy) expand(bindings *env) yamly {
	result, ok := expandStr(string(x), bindings)
	if !ok {
		return interpolate(x, bindings)
	}
	if resultStr, ok := result.(stringy); ok {
		return interpolate(resultStr.expand(bindings), bindings)
	}
	return result.expand(bindings)
}

func expandMapKey(kstr stringy, bindings *env) (yamly, bool) {
	if strings.HasPrefix(string(kstr), "^") {
		variableName := kstr[1:]
		value, ok := bindings.lookup(variableName)
		if !ok {
			panic(fmt.Sprintf("ERROR: Variable ^%v is not defined", variableName))
		}
		if !reflect.ValueOf(value).Type().Comparable() {
			panic(fmt.Sprintf("Unable to use %v as map key, '%v' is not comparable.", variableName, value))
		}
		return value, true
	}
	return nil, false
}

func (m mapy) expand(bindings *env) yamly {

	newdict := mapy{}

	// Lookahead for functions. Some have Lazy maps we dont want to expand yet...
	if ok, function, rhs := isFunction(m, bindings); ok {
		log.Printf("isFunction returned => :\n    %v\n    %#v\n    %v\n", ok, function, rhs)
		var actualArgs, expandedResult yamly

		if function.isEager() {
			actualArgs = rhs.expand(bindings)
		} else { // lazy, quote
			actualArgs = rhs
		}
		expandedResult = function.apply(m, actualArgs, bindings)
		if function.isQuote() {
			return expandedResult
		}
		return expandedResult.expand(bindings)
	}
	// Just a normal map - not a function
	for k, v := range m {

		if kstr, ok := k.(stringy); ok {
			keyv, ok := expandMapKey(kstr, bindings)
			if ok {
				if _, ok := newdict[keyv]; ok {
					panic(fmt.Sprintf("ERROR: duplicate map key '%v' computed from '%v' in %v", keyv, kstr, m))
				}
				newdict[keyv] = v.expand(bindings)
				continue
			}
		}
		interpKey := interpolate(k, bindings)
		if interpKey != k {
			// string containing {{ }} - only these keys are expanded
			if _, ok := newdict[interpKey]; ok {
				panic(fmt.Sprintf("ERROR: duplicate map key '%v' in %v", interpKey, m))
			}
			newdict[interpKey] = v.expand(bindings)
			continue
		} else {
			log.Printf("not interpolating key %#v", k)
		}
		if _, ok := newdict[k]; ok {
			panic(fmt.Sprintf("ERROR: duplicate map key '%v' in %v", interpKey, m))
		}
		expanded := v.expand(bindings)
		if _, ok := expanded.(empty); !ok { // Skip golang.EMPTY values
			newdict[k] = expanded
		}
	}
	return newdict
}

func expandFile(filename string, bindings *env) error {
	log.Printf("expandFile: filename %v\n", filename)

	var dirForThisFile, path, priorDir string
	var err error
	priorFile, ok := bindings.lookup(stringy("__FILE__"))
	if !ok {
		priorFile = stringy("")
	}
	d, ok := bindings.lookup(stringy("__DIR__"))
	if !ok {
		log.Printf("expandFile: __DIR__ => not bound\n")
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(pwd)
		if err != nil {
			return err
		}
		dirForThisFile = filepath.Dir(abs)
	} else {
		priorDir = string(d.(stringy))
	}

	if filename == "-" {
		path = "-"
		dirForThisFile = "."
	} else if strings.HasPrefix(filename, "/") {
		path = filename
		dirForThisFile = filepath.Dir(path)
	} else {
		path, err = filepath.Abs(filepath.Join(priorDir, filename)) // resolve relative paths
		if err != nil {
			return err
		}
		dirForThisFile = filepath.Dir(path)
	}
	bindings.bind["__DIR__"] = stringy(dirForThisFile)
	bindings.bind["__FILE__"] = stringy(filepath.Base(path))
	log.Printf("expandFile: new __DIR__ => %v\n", dirForThisFile)
	log.Printf("expandFile: new __FILE__ => %v\n", filepath.Base(path))

	input, err := os.Open(path)
	if err != nil {
		log.Printf("open => %v %v %v\n", path, input, err)
		return err
	}
	err = expandStream(input, filename, bindings)
	if err != nil {
		return err
	}

	bindings.bind["__DIR__"] = stringy(priorDir) // restore prior file
	bindings.bind["__FILE__"] = priorFile
	return nil
}

func expandStream(input io.Reader, _ string, bindings *env) (err error) {
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
		// fmt.Println(expanded)
		if _, eok := expanded.(empty); eok {
			log.Printf("expandSteam: doc is empty\n")
			if documentCount > 0 {
				documentCount++
			}
			continue
		}
		if _, eok := expanded.(nily); eok {
			log.Printf("expandSteam: doc is nil\n")
			if documentCount > 0 {
				documentCount++
			}
			continue
		}
		if array, ok := expanded.(seqy); ok {
			if len(array) == 0 {
				log.Printf("expandSteam: doc is empty list\n")
				if documentCount > 0 {
					documentCount++
				}
				continue
			}
		}
		if mapz, ok := expanded.(mapy); ok {
			if len(mapz) == 0 {
				log.Printf("expandSteam: doc is empty map\n")
				if documentCount > 0 {
					documentCount++
				}
				continue
			}
		}
		if bindings.engine.outFormat == YAML {
			enc := yaml.NewEncoder(bindings.engine.output)
			enc.SetIndent(2)
			if documentCount > 0 {
				_, _ = fmt.Fprintln(bindings.engine.output, "---")
			}
			err = enc.Encode(expanded.declassify())
			if err != nil {
				return err
			}
		} else if bindings.engine.outFormat == LINES {
			switch s := expanded.(type) {
			case seqy:
				for _, item := range s {
					fmt.Println(item.String())
				}
			default:
				fmt.Println(s.String())
			}

		} else {
			jsonTree := expanded.declassify(JSON)
			jenc := json.NewEncoder(bindings.engine.output)
			jenc.SetIndent("", "  ")
			err = jenc.Encode(jsonTree)
			if err != nil {
				log.Printf("expandSteam: json.Encode %v", err)
				return err
			}
		}
		documentCount++
	}
}
