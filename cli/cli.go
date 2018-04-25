package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	pb "github.com/andefined/go-chat-cli/service"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:50051", "Server Address (host:port)")
)

// Out Sends Messages to the Streaming Service
func Out(ch chan pb.Message, reader *bufio.Reader, authorName string) {
	for {
		msg, _ := reader.ReadString('\n')
		ch <- pb.Message{
			Author: &pb.Author{
				Name: authorName,
			},
			Body: msg,
		}
	}
}

// In Receives Messages from the Streaming Service
func In(stream pb.Chat_StreamClient, ch chan pb.Message) {
	for {
		msg, _ := stream.Recv()
		ch <- *msg
	}
}

func main() {
	// Parse CLI Flags
	flag.Parse()

	// Create the gRPC Service
	// Parse Server Options
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	// Connect to Server
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("Fail to Dial: %v", err)
	}
	defer conn.Close()

	// Create gRPC Chat Client
	client := pb.NewChatClient(conn)
	// Connect to Stream
	stream, err := client.Stream(context.Background())
	if err != nil {
		log.Fatalf("Stream Connection Failed: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\x1b[29;1mWelcome to gRPC Chat CLI!\x1b[0m")
	fmt.Print("\x1b[30;1mPlease enter your username: \x1b[0m")

	authorName, _ := reader.ReadString('\n')
	truncAName := truncateString(authorName, 12)

	if err = stream.Send(&pb.Message{Author: &pb.Author{Name: authorName}}); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s > ", "\x1b[32;1m"+truncAName+"\x1b[0m")

	incomingMessages := make(chan pb.Message, 100)
	go In(stream, incomingMessages)

	outgoingMessages := make(chan pb.Message, 100)
	go Out(outgoingMessages, reader, authorName)

	for {
		select {
		case o := <-outgoingMessages:
			username := truncateString(authorName, 12)
			username = "\x1b[32;1m" + username + "\x1b[0m"
			fmt.Printf("%s > ",
				username,
			)
			stream.Send(&o)

		case i := <-incomingMessages:
			username := truncateString(i.Author.Name, 12)
			if i.Author.Name != authorName {
				username = "\x1b[33;1m" + username + "\x1b[0m"
			} else {
				username = "\x1b[32;1m" + username + "\x1b[0m"
			}
			fmt.Printf("%s > %s",
				username,
				i.Body,
			)
		}
	}
}

func truncateString(s string, i int) string {
	if len(s) > i {
		s = s[0:i] + "..."
	}
	return strings.Trim(s, "\r\n")
}
