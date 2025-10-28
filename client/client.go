package main

import (
	p "Chit-Chat/proto"
	"bufio"
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// todo: change log messages to correct form
func ListenForMessages(stream grpc.BidiStreamingClient[p.Message, p.Message]) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Fatalf("Lost connection to server: %v", err)
		}

		switch msg.MsgType {
		case 0:
			log.Printf("%v - [SERVER]: %s has joined the chat!", msg.GetTimestamp(), msg.GetUsername())
			break
		case 1:
			log.Printf("%v - [%s]: %s", msg.GetTimestamp(), msg.GetUsername(), msg.GetText())
			break
		case 2:
			log.Printf("%v - [SERVER]: %s has left the chat!", msg.GetTimestamp(), msg.GetUsername())
			break
		}
	}
}

func main() {
	conn, err := grpc.NewClient(":8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := p.NewChatServiceClient(conn)

	stream, err := c.ChatStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	go ListenForMessages(stream)
	fmt.Println("Enter a username: ")
	in := bufio.NewScanner(os.Stdin)
	in.Scan()
	u := in.Text()

	stream.Send(&p.Message{MsgType: 0, Username: u, Text: "", Timestamp: 0})

	fmt.Printf("Welcome %s, say hello in the chat!\n", u)
	for in.Scan() {
		text := in.Text()
		stream.Send(&p.Message{MsgType: 1, Username: u, Text: text, Timestamp: 0})
	}
}
