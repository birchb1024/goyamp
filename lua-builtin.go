package goyamp

import (
	"fmt"
	"log"
	"math"

	lua "github.com/yuin/gopher-lua"
)

// Cannot us Lua nil because that implies empty in tables. So use this singleton object instead
var luanily = lua.LUserData{Value: "NILY", Env: nil, Metatable: nil}

func (x nily) gopherluaify(*lua.LState) lua.LValue     { return &luanily }
func (e empty) gopherluaify(*lua.LState) lua.LValue    { return lua.LString(e.String()) }
func (x inty) gopherluaify(*lua.LState) lua.LValue     { return lua.LNumber(x) }
func (x float64y) gopherluaify(*lua.LState) lua.LValue { return lua.LNumber(x) }
func (x booly) gopherluaify(*lua.LState) lua.LValue    { return lua.LBool(x) }
func (x stringy) gopherluaify(*lua.LState) lua.LValue  { return lua.LString(x) }
func (m mapy) gopherluaify(L *lua.LState) lua.LValue {
	result := L.CreateTable(len(m), len(m))
	result.Metatable = L.GetGlobal("mapy")
	for k, v := range m {
		result.RawSet(lua.LString(k.String()), v.gopherluaify(L))
	}
	return result
}

func (s seqy) gopherluaify(L *lua.LState) lua.LValue {
	result := L.CreateTable(len(s), len(s))
	result.Metatable = L.GetGlobal("seqy")
	for i, v := range s {
		result.Insert(i+1, v.gopherluaify(L))
	}
	return result
}
func (r macroFunction) gopherluaify(*lua.LState) lua.LValue    { return lua.LString(r.String()) }
func (r compiledFunction) gopherluaify(*lua.LState) lua.LValue { return lua.LString(r.String()) }
func (x unknowny) gopherluaify(*lua.LState) lua.LValue         { return lua.LString(x.String()) }

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

		if float64(x) == float64(math.Trunc(float64(x))) {
			return inty(x)
		} else {
			return float64y(x)
		}
	case lua.LBool:
		return booly(x)
	case lua.LString:
		return stringy(x)
	case *lua.LTable:
		allInts := true
		keys := map[int]bool{}
		closed := true
		const MaxUint = ^uint(0)
		const MaxInt = int(MaxUint >> 1)
		minKey := MaxInt
		maxKey := 0
		length := 0

		// Scan the table
		L.ForEach(x, func(k lua.LValue, v lua.LValue) {
			length += 1
			if ln, ok := k.(lua.LNumber); ok {
				i := int(ln)
				keys[i] = true
				if i < minKey {
					minKey = i
				}
				if i > maxKey {
					maxKey = i
				}
			} else {
				allInts = false
			}
		})
		if length == 0 { // Empty
			if x.Metatable == L.GetGlobal("mapy") {
				return mapy{}
			} else {
				return seqy{}
			}
		}
		if allInts { // Look for gaps in the key sequence
			for i := minKey; i <= maxKey; i++ {
				if _, ok := keys[i]; !ok {
					closed = false
					break
				}
			}
		}
		if allInts && closed {
			log.Printf("makeseqy %d %d %d", minKey, maxKey, maxKey-minKey+1)
			result := make(seqy, maxKey-minKey+1)
			for i := minKey; i <= maxKey; i++ {
				v := x.RawGetInt(i)
				result[i-1] = classifyLua(L, v)
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

	// Add additional types
	L.SetGlobal("nily", &luanily)
	for _, mtn := range []string{"seqy", "mapy"} {
		mt := L.NewTypeMetatable(mtn)
		L.SetGlobal(mtn, mt)
		if err := L.DoString(fmt.Sprintf("%[1]s['name']='%[1]s'; ", mtn)); err != nil {
			panic(fmt.Sprintf("gopherlua:  %s", err))
		}
	}
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
