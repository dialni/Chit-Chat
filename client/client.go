package main

import (
	p "Chit-Chat/proto"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ListenForMessages(stream grpc.BidiStreamingClient[p.Message, p.Message]) {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Fatalf("failed to receive a message: %v", err)
		}
		fmt.Printf("%s says: %s\n", "Server", msg.Text)
	}
}

func main() {
	conn, err := grpc.NewClient(":8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := p.NewChatServiceClient(conn)
	msg := p.Message{Text: "Hello from client!"}

	resp, err := c.SayHello(context.Background(), &msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Response from server: ", resp.Text)
	stream, err := c.ChatStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	go ListenForMessages(stream)
	fmt.Println("Enter a username: ")
	in := bufio.NewScanner(os.Stdin)
	in.Scan()
	uname := in.Text()

	fmt.Printf("Welcome %s, say hello in the chat!\n", uname)
	for in.Scan() {
		text := in.Text()
		stream.Send(&p.Message{Text: text})
	}
}
