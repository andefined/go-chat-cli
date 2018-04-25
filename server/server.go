package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pb "github.com/andefined/go-chat-cli/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "Port to Listen")
)

// ChatHandler ...
type ChatHandler struct {
	cachedUsers map[string]chan pb.Message
	mu          sync.Mutex
}

// Listen ...
func (p *ChatHandler) Listen(stream pb.Chat_StreamServer, ch chan<- pb.Message) {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		ch <- *req
	}
}

// Filter ...
func (p *ChatHandler) Filter(author string, m pb.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for receiver, q := range p.cachedUsers {
		if author != receiver {
			q <- m
		}
	}
}

// Stream ...
func (p *ChatHandler) Stream(stream pb.Chat_StreamServer) error {
	// Open Streamer
	s, err := stream.Recv()
	if err != nil {
		log.Fatalf("Failed to Stream: %v", err)
	}

	// Check whether or not user messages channel allready exists
	if _, exists := p.cachedUsers[s.Author.Name]; exists {
		return fmt.Errorf("Username allready exists")
	}

	// Create user messages channel
	p.cachedUsers[s.Author.Name] = make(chan pb.Message, 100)

	// Non-Blocking Client Messages Channel
	messagesChannel := make(chan pb.Message, 100)
	go p.Listen(stream, messagesChannel)

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case clientMessage := <-messagesChannel:
			go p.Filter(s.Author.Name, clientMessage)
		case channelMessage := <-p.cachedUsers[s.Author.Name]:
			stream.Send(&channelMessage)
		}
	}
}

func main() {
	// Parse CLI Flags
	flag.Parse()

	// Listen Server
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to Listen: %v", err)
	}
	// Non-Blocking Kill Channel
	errChanTCP := make(chan os.Signal, 10)
	signal.Notify(errChanTCP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-errChanTCP
		if i, ok := s.(syscall.Signal); ok {
			os.Exit(int(i))
		} else {
			os.Exit(0)
		}
	}()
	// Create the gRPC Service
	// Parse Server Options
	var opts []grpc.ServerOption
	svc := grpc.NewServer(opts...)
	// Register Service Handlers
	pb.RegisterChatServer(svc, &ChatHandler{
		cachedUsers: make(map[string]chan pb.Message),
	})
	log.Printf("Starting gRPC Server on: :%v", *port)
	// Register reflection service on gRPC server.
	//
	// gRPC Server Reflection provides information about publicly-accessible
	// gRPC services on a server, and assists clients at runtime
	// to construct RPC requests and responses without precompiled service information.
	// It is used by gRPC CLI, which can be used to introspect server protos
	// and send/receive test RPCs.
	//
	// https://github.com/grpc/grpc-go/blob/master/Documentation/server-reflection-tutorial.md
	reflection.Register(svc)
	// Serve gRPC Service with Error
	errChanSVC := make(chan error, 10)

	go func() {
		errChanSVC <- svc.Serve(listen)
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChanSVC:
			if err != nil {
				log.Fatal(err)
			}
		case s := <-signalChan:
			log.Println(fmt.Sprintf("Captured message %v. Exiting...", s))
			os.Exit(0)
		}
	}
}
