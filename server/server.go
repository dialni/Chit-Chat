package main

import (
	p "Chit-Chat/proto"
	"context"
	"io"
	"log"
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	p.UnimplementedChatServiceServer
}

func (s *Server) SayHello(ctx context.Context, msg *p.Message) (*p.Message, error) {
	log.Printf("Server Received: %s", msg)
	return &p.Message{Text: "Hello from server!\n"}, nil
}

func (s *Server) ChatStream(stream p.ChatService_ChatStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		log.Printf("Server Received: %s", msg.Text)
		if err := stream.Send(&p.Message{Text: "Greetings, traveler!"}); err != nil {
			return err
		}
	}
}

func main() {
	s := &Server{}
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("server listening at %v", lis.Addr())
	p.RegisterChatServiceServer(grpcServer, s)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
