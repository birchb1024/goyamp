-- Upercase all items in a YAML tree
function uppertree(t)
    tt = type(t)
    if tt == "nil" then
      return t
    elseif tt == "string" then
      return string.upper(t)
    elseif tt == "table" then
      local result = {}
      for k,v in pairs(t) do
        result[string.upper(k)] = uppertree(v)
      end
      return result
    else
      return t
    end
end

-- Execute an external command and capture and return the stdout
function os.capture(cmd)
  local f = assert(io.popen(cmd, 'r'))
  local s = assert(f:read('*a'))
  f:close()
  return s
end

