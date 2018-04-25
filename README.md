# Go gRPC Chat CLI

A Chat Server/CLI implementation, using gRPC and golang.

## Installation
```bash
# clone the repo
git clone git@github.com:andefined/go-chat-cli.git
# install dependencies
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/protoc-gen-go
# generate grpc client/server
make build
```
## Usage
```bash
# go-chat-server
Usage of go-chat-server:
    -port int
        Port to Listen (default 50051)
# go-chat
Usage of go-chat:
    -server_addr string
        Server Address (host:port) (default "127.0.0.1:50051")
```
## Development
```bash
# start the server
go run server/server.go --port 50051
# start the client
go run cli/cli.go --server_addr 127.0.0.1:50051
```
## Build
```bash
# go-chat-server
go build -o dist/go-chat-server server/server.go
# go-chat
go build -o dist/go-chat cli/cli.go
```
