-- Uppercase all strings in a YAML tree - atomic keys
function uppertree(t)
    local tt = type(t)
    if tt == "string" then
      return string.upper(t)
    elseif tt == "table" then
      local k, v = next(t, nil)
      local result = {}
      while k do
        if type(k) == "string" then
          result[string.upper(k)] = uppertree(v)
        else
          result[k] = uppertree(v)
        end
          k, v = next(t, k)
      end
      return result
    else
      return t
    end
end