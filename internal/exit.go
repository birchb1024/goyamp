package internal

import (
	"os"
	"strconv"
)

//
// Exit the current processing and exit the process cleanly.
//
func exitBuiltin(tree mapy, args yamly, bindings *env) yamly {
	result := 1
	switch a := args.(type) {
	case empty:
		result = 0
	case nily:
		result = 0
	case inty:
		result = int(a)
	case stringy:
		if i, err := strconv.Atoi(string(a)); err == nil {
			result = i
		}
	}

	os.Exit(result)
	return nily{}
}
