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
// ListenForMessages is mostly self-explanitory, it listens are parses messages from the server to the clients terminal
func ListenForMessages(stream grpc.BidiStreamingClient[p.Message, p.Message]) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Fatalf("[CLIENT]: Lost connection to server.")
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

	// Start listening for messages in the background.
	go ListenForMessages(stream)
	fmt.Println("Enter a username: ")
	in := bufio.NewScanner(os.Stdin)
	in.Scan()
	u := in.Text()

	// Send a "I have joined the chat" message to the server, which is broadcasted by the server to everyone
	stream.Send(&p.Message{MsgType: 0, Username: u, Text: "", Timestamp: 0})

	// Inform the user, that the client is ready for input
	fmt.Printf("Welcome %s, say hello in the chat!\n", u)
	for in.Scan() {
		text := in.Text()
		stream.Send(&p.Message{MsgType: 1, Username: u, Text: text, Timestamp: 0})
	}
}
