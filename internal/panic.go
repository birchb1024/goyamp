package internal

// Fail the current processing and exit the process.
func panicBuiltin(tree mapy, args yamly, bindings *env) yamly {
	panic(args)
	// Compiler knows we're paniced so no return is required.
}
