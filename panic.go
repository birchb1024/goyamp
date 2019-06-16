package goyamp

//
// Fail the current processing and exit the process.
//
func panic_builtin(tree mapy, args yamly, bindings *env) yamly {
	panic(args)

	return nily{}
}
