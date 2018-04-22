protoc:
	go get -u github.com/golang/protobuf/protoc-gen-go
	protoc -I. -I${GOPATH}/src --go_out=plugins=grpc:. service/chat.proto

cli:
	go get google.golang.org/grpc
	go get github.com/andefined/go-chat-cli/service
	go build -o \
	    $GOPATH/src/github.com/andefined/go-chat-cli/dist/go-chat \
	    $GOPATH/src/github.com/andefined/go-chat-cli/cli/cli.go

server:
	go get google.golang.org/grpc
	go get github.com/andefined/go-chat-cli/service
	go build -o \
	    $GOPATH/src/github.com/andefined/go-chat-cli/dist/go-chat \
	    $GOPATH/src/github.com/andefined/go-chat-cli/server/server.go

build: protoc cli server
