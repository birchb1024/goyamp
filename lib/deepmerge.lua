#!/usr/bin/env glua
module("dm", package.seeall)

_ = [[ 
This file contains the deepmerge(a, b) function. 

Deepmerge takes two objects A and B, normally expected to be tables containing lists and maps.
It scans the tables and based on the kind of table it merges the two together and returns the merge. 
If there is a conflict A always takes priority over B. Conflicts occur if two maps have the same key, 
or there are differing types. Let us see the different cases:

Maps with keys

    { a = 1, c = 4, d = {y = 10} }, { a = 2, b = 3, d = { y = 99, u = 100 } }

    {
        a = 1,
        b = 3,
        c = 4,
        d = {
          u = 100,
          y = 10
        }
      }
      

Arrays (sequential integer keys with no gaps):

Bags (where there are duplicate items)

    One list is added to the end of the other.

    { 1, 1, 2, 3 }, {3, 4, 5, 1}

    { 1, 1, 2, 3, 3, 4, 5, 1}

Sets (where both A and B have unique elements) 

    A set union is returned

    { 1, 2, 3 }, {3, 4, 5, 1}

    { 1, 2, 3, 4, 5}

]]

-- --Debug functions, uncomment when needed
-- local inspect = require('inspect')
-- ix = function (x) print(inspect.inspect(x)) end

function classify(t)
    if type(t) ~= "table" then return type(t) end  
    -- Determine if a Lua table is an array. https://ericjmritz.wordpress.com/2014/02/26/lua-is_array/
    local i = 0
    for _,_ in pairs(t) do
        i = i + 1
        if t[i] == nil then return "map" end
    end
    -- it's an array
    local set = {}
    for _, v in ipairs(t) do
		if set[v] ~= nil then return "array" end
		set[v] = true
		end
    return "set"
end

function set_merge(a, b)
	-- Treat arrays as sets. 
	-- result randomly ordered
	local elements = {}
	for i, v in ipairs(a) do
		elements[v] = true
	end
	for i, v in ipairs(b) do
		elements[v] = true
	end
	local results = {}
	for k, _ in pairs(elements) do
		table.insert(results, k)
	end
	return results
end

function array_merge(a, b)
	-- Treat arrays as a bag, ie possibly with duplicates and ordered.
	-- result ordered a first, then b
	local result = {}
	for i, v in ipairs(a) do
		result[i] = v
	end
	for i, v in ipairs(b) do
		table.insert(result, v)
	end
	return result
end

function map_merge(a, b)
	-- map elements with same keys to be merged, otherwies just added
	local result = {}
	for ka, va in pairs(a) do
		if b[ka] ~= nil then -- both maps have this key 
			result[ka] = deep_merge(va, b[ka])
		else
			result[ka] = va
		end
	end

	for kb, vb in pairs(b) do
		if a[kb] == nil then    -- A does not have this key
			result[kb] = vb       -- For a hard earned thirst.
		else
		-- A already has this key and it trumps B
		end
	end
	return result
end

function isarray(typ)
	arraytypes = {set = true, array = true}
	return arraytypes[typ] ~= nil
end
--
-- Deep merge two trees 
--
function dm.deep_merge(a, b)
    --ix({"deep_merge: ", a, b})
    local at = classify(a)
    local bt = classify(b)
    
    if at == "number" or at == "string" then
    	return a
    end
   
	if at == bt then
		if at == "set" then
	        return set_merge(a, b)
    	elseif at == "array" then
        	return array_merge(a, b)
    	elseif at == "map" then
        	return map_merge(a, b)
		else
			return a
		end
	else
	    if isarray(at) and isarray(bt) then	
			return array_merge(a, b)
		else
			return a
		end
	end
end

return dm

--ix(deep_merge({ a = 1, c = 4, d = {y = 10} }, { a = 2, b = 3, d = { y = 99, u = 100 } }))
--ix(deep_merge({ 1, 1, 2, 3 }, {3, 4, 5, 1}))
--ix(deep_merge({ 1, 2, 3 }, {3, 4, 5, 1}))