package goyamp

//
// Fail the current processing and exit the process.
//
func panic_builtin(tree mapy, args yamly, bindings *env) yamly {
	panic(args)
	// Compiler knows we're paniced so no return is requried.
}
