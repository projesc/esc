
log(foo)

log("CFG "..config("foo"))
log("CFG "..config("fuz"))

on("*","test",function(msg)
  log("Received "..msg.name)

  send(self(),"hello","world")
end)

on("*","hello",function(msg)
  log("Hello "..msg.payload.."!")
end)

i=0
tick(2,function()
  log("Sending hello "..i.."!")
  i = i +1
  send(self(),"test","hello")
  return i <= 5
end)

