
log("Internal hello")

local json = require("json")
local http = require("http")

resp, err = http.get("https://jsonplaceholder.typicode.com/users")
log("<->"..json.decode(resp.body)[1].company.bs)

