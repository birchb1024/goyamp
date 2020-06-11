package goyamp

import (
	"fmt"
	"log"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

func (x nily) gopherluaify(L *lua.LState) lua.LValue     { return lua.LString(x.String()) }
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
		result.RawSet(lua.LNumber(i), v.gopherluaify(L))
	}
	return result
}
func (r macroFunction) gopherluaify(L *lua.LState) lua.LValue    { return lua.LString(r.String()) }
func (r compiledFunction) gopherluaify(L *lua.LState) lua.LValue { return lua.LString(r.String()) }
func (x unknowny) gopherluaify(L *lua.LState) lua.LValue         { return lua.LString(x.String()) }

func classifyLua(L *lua.LState, x lua.LValue) yamly {
	switch x := x.(type) {
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

		L.ForEach(x, func(k lua.LValue, v lua.LValue) {
			if i, err := strconv.Atoi(k.String()); err == nil {
				keys[i] = true
				//				fmt.Println("key: ", k, k.Type(), i, v)
			} else {
				allInts = false
			}
		})
		if allInts {
			for i := 0; i < len(keys); i++ {
				if _, ok := keys[i]; !ok {
					closed = false
					break
				}
			}
		}
		if allInts && closed {
			result := make(seqy, len(keys))
			L.ForEach(x, func(k lua.LValue, v lua.LValue) {
				if i, err := strconv.Atoi(k.String()); err == nil {
					//					fmt.Println("key: ", k, k.Type(), i, v)
					result[i] = classifyLua(L, v)
				} else {
					panic(fmt.Sprintf("gopherlua: bad index %v in '%v", k, x))
				}
			})
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
	//
	if err := L.DoFile("init.lua"); err != nil {
		panic(fmt.Sprintf("gopherlua eror in init.lua: %s", err))
	}

	L.SetGlobal("args", tb)
	if err := L.DoString(s); err != nil {
		panic(fmt.Sprintf("gopherlua eror: %s", err))
	}

	// Process the response from the sub-process
	//
	return classifyLua(L, L.GetGlobal("result"))
}
