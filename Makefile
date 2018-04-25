get:
	go get -u github.com/golang/protobuf/protoc-gen-go
protoc:
	protoc -I. -I${GOPATH}/src --go_out=plugins=grpc:. service/chat.proto
build: get protoc
