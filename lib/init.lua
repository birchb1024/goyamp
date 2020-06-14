
-- Execute an external command and return the stdout
function os.capture(cmd)
  local f = assert(io.popen(cmd, 'r'))
  local s = assert(f:read('*a'))
  f:close()
  return s
end

-- local info = debug.getinfo(1,'S')
-- print(info.source, " Loaded init.lua")
-- print(info.source, " package.path: ", package.path)
-- print(info.source, " executable_directory: ", executable_directory)
