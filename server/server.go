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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var (
	tls      = flag.Bool("tls", false, "Use TLS")
	certFile = flag.String("cert_file", "", "Path to Cert File")
	keyFile  = flag.String("key_file", "", "Path to Key File")
	port     = flag.Int("port", 50051, "Port to Listen")
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
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		s := <-ch
		if i, ok := s.(syscall.Signal); ok {
			os.Exit(int(i))
		} else {
			os.Exit(0)
		}
	}()
	// Create the gRPC Service
	// Parse Server Options
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = ""
		}
		if *keyFile == "" {
			*keyFile = ""
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to Generate Credentials: %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	svc := grpc.NewServer(opts...)
	// Register Service Handlers
	pb.RegisterChatServer(svc, &ChatHandler{
		cachedUsers: make(map[string]chan pb.Message),
	})

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
	log.Printf("Listening Serice on: %v", *port)
	if err := svc.Serve(listen); err != nil {
		log.Fatalf("Failed to Serve: %v", err)
	}
}
