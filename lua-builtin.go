package goyamp

import (
	"fmt"
	"log"

	lua "github.com/yuin/gopher-lua"
)

var luanily = lua.LUserData{"NILY", nil, nil}

func (x nily) gopherluaify(L *lua.LState) lua.LValue     { return &luanily }
func (e empty) gopherluaify(L *lua.LState) lua.LValue    { return lua.LString(e.String()) }
func (x inty) gopherluaify(L *lua.LState) lua.LValue     { return lua.LNumber(x) }
func (x float64y) gopherluaify(L *lua.LState) lua.LValue { return lua.LNumber(x) }
func (x booly) gopherluaify(L *lua.LState) lua.LValue    { return lua.LBool(x) }
func (x stringy) gopherluaify(L *lua.LState) lua.LValue  { return lua.LString(x) }
func (m mapy) gopherluaify(L *lua.LState) lua.LValue {
	result := L.CreateTable(len(m), len(m))
	for k, v := range m {
		result.RawSet(lua.LString(k.String()), v.gopherluaify(L))
	}
	return result
}

func (s seqy) gopherluaify(L *lua.LState) lua.LValue {
	result := L.CreateTable(len(s), len(s))
	for i, v := range s {
		// fmt.Println("Insert", i+1, v.gopherluaify(L))
		result.Insert(i+1, v.gopherluaify(L))
	}
	return result
}
func (r macroFunction) gopherluaify(L *lua.LState) lua.LValue    { return lua.LString(r.String()) }
func (r compiledFunction) gopherluaify(L *lua.LState) lua.LValue { return lua.LString(r.String()) }
func (x unknowny) gopherluaify(L *lua.LState) lua.LValue         { return lua.LString(x.String()) }

func classifyLua(L *lua.LState, x lua.LValue) yamly {
	switch x := x.(type) {
	case *lua.LUserData:
		if x == &luanily {
			return nily{}
		}
		log.Printf("not classified %#v", x)
		return unknowny{x: x}

	case *lua.LNilType:
		return nily{}
	case lua.LNumber:
		return float64y(x)
	case lua.LBool:
		return booly(x)
	case lua.LString:
		return stringy(x)
	case *lua.LTable:
		allInts := true
		keys := map[int]bool{}
		closed := true
		minKey := 10000000000
		maxKey := 0

		L.ForEach(x, func(k lua.LValue, v lua.LValue) {
			if ln, ok := k.(lua.LNumber); ok {
				i := int(ln)
				keys[i] = true
				if i < minKey {
					minKey = i
				}
				if i > maxKey {
					maxKey = i
				}
				//				fmt.Println("key: ", k, k.Type(), i, v)
			} else {
				allInts = false
			}
		})
		if maxKey == 0 { // Empty table!
			allInts = false
		}
		if allInts {
			for i := minKey; i <= maxKey; i++ {
				if _, ok := keys[i]; !ok {
					closed = false
					break
				}
			}
		}
		// fmt.Println("allInts closed: ", allInts, closed, minKey, maxKey, x)
		// L.ForEach(x, func(k lua.LValue, v lua.LValue) {
		// 	fmt.Println("                ", k.String(), v.String())
		// })
		if allInts && closed {
			log.Printf("makeseqy %d %d %d", minKey, maxKey, maxKey-minKey+1)
			result := make(seqy, maxKey-minKey+1)
			for i := minKey; i <= maxKey; i++ {
				// fmt.Println("resultB", i, result)
				v := x.RawGetInt(i)
				result[i-1] = classifyLua(L, v)
				// fmt.Println("resultA", result)
			}
			return result
		}
		result := mapy{}
		L.ForEach(x, func(k lua.LValue, v lua.LValue) {
			result[classifyLua(L, k)] = classifyLua(L, v)
		})
		return result

	default:
		log.Printf("not classified %#v", x)
		return unknowny{x: x}
	}
}

func gopherluaBuiltin(tree mapy, args yamly, bindings *env) yamly {
	log.Printf("gopherlua: %v", args)

	assertSingleKey(tree)
	argsmap, ok := args.(mapy)
	if !ok {
		panic(fmt.Sprintf("execute: '%v' is not a valid args map", args))
	}

	assertKeys(map[string]bool{"args": true, "script": true}, argsmap)
	a := argsmap[stringy("args")]
	s := argString(tree, argsmap, "script", "")



	L := lua.NewState()
	defer L.Close()
	// // Convert argument to Lua format
	//
	tb := a.gopherluaify(L)
	// Invoke Lua
	//    	First set the package.path to avoid disappointment.
	// 		The default gopher-lua loads from /usr/local and we dont want random effects,
	//		so we limit loading to the goyamp context.
	//		LUA_PATH is always there if needed
	// 		Always Add the directory holding this script to the Lua search path
	script_dir := ""
	if sy, ok := bindings.lookup(stringy("__DIR__")); ok {
		sd, ok := sy.(stringy)
		script_dir = string(sd)
		if !ok {
			panic(fmt.Sprintf("Was expecting __DIR__ to be a string, but got %v", script_dir))
		}
	}

	lua_path_setup := `
		executable_directory = "` + goyampExecutablePath + `"
		__DIR__ = "` + script_dir + `"
		if os.getenv("LUA_PATH") ~= nil then
			package.path = os.getenv("LUA_PATH")
		else
			package.path = "./?.lua;./?.lc;"..executable_directory.."/lib/?.lua;"..executable_directory.."/lib/?.lc"
		end
		if __DIR__ ~= "" then
			package.path = __DIR__.."/?.lua;"..package.path
		end`
	if err := L.DoString(lua_path_setup); err != nil {
		panic(fmt.Sprintf("gopherlua:  %s", err))
	}
	// Run init if it exists - fail silently
	if err := L.DoString(`require('init')`); err != nil {
		log.Printf("Warning: gopherlua error in init.lua, continuing: %s\n", err)
	}
	// Now actually run the script string
	L.SetGlobal("args", tb)
	if err := L.DoString(s); err != nil {
		panic(fmt.Sprintf("gopherlua eror: %s", err))
	}

	// Process the response from the other interpreter
	//
	r := L.GetGlobal("result")
	log.Printf("r: %v", r)
	cr := classifyLua(L, r)
	log.Printf("cr: %v", cr)

	return cr
}
