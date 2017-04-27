
all: clean build

build: esc-amd64 esc-arm

esc-amd64: 
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o esc-amd64 cmd/esc/main.go

esc-arm: 
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=6 go build -a -tags netgo -ldflags '-w' -o esc-arm cmd/esc/main.go

docker: esc-amd64
	docker build -t diogok/esc .

docker-arm: esc-arm
	docker build -t diogok/esc:arm -f Dockerfile.arm .

clean:
	rm -f esc-*

deps:
	go get github.com/ghodss/yaml
	go get github.com/micro/mdns
	go get github.com/diogok/gorpc
	go get github.com/patrickmn/go-cache
	go get github.com/yuin/gopher-lua
	go get layeh.com/gopher-json
	go get github.com/cjoudrey/gluahttp

push:
	docker push diogok/esc

run:
	go run -race cmd/esc/main.go

install:
	go install cmd/esc/main.go
