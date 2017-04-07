
all: clean build

build: dsc-amd64 dsc-arm

dsc-amd64: 
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o dsc-amd64

dsc-arm: 
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -a -tags netgo -ldflags '-w' -o dsc-arm

docker: dsc-amd64
	docker build -t diogok/dsc .

docker-arm: dsc-arm
	docker build -t diogok/dsc:arm -f Dockerfile.arm .

run:
	go run *.go

clean:
	rm -f dsc-amd64
	rm -f dsc-arm

deps:
	go get github.com/ghodss/yaml
	go get github.com/micro/mdns
	go get github.com/diogok/gorpc
	go get github.com/patrickmn/go-cache

push:
	docker push diogok/dsc
