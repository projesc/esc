
all: clean build

build: dsc-amd64

dsc-amd64: 
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o dsc-amd64

docker: dsc-amd64
	docker build -t diogok/dsc .

run:
	go run *.go

clean:
	rm -f dsc-amd64

deps:
	go get github.com/ghodss/yaml
	go get github.com/hashicorp/mdns
	go get github.com/valyala/gorpc
