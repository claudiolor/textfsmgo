buildargs = -o dist/textfsmgo ./pkg/main/main.go
build:
	go build $(buildargs)

install:
	go build $(buildargs)
	cp ./dist/textfsmgo $(shell go env GOPATH)/bin/textfsmgo

test:
	go test -v ./...

cover:
	go test ./... -cover

