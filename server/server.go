package main

import (
	p "Chit-Chat/proto"
	"log"
	"net"

	"google.golang.org/grpc"
)

type BrokerMessage struct {
	MsgType int
	User    string
	Text    string
	Stream  *p.ChatService_ChatStreamServer
}

type Server struct {
	p.UnimplementedChatServiceServer
	clients    []p.ChatService_ChatStreamServer
	brokerChan chan BrokerMessage
}

// todo: change log messages to correct form
// MessageBroker is the main logic behind broadcasting data and keeping track of the logical time (Lamport timestamp).
// This architecture was chosen, as it can handle multiple concurrent requests, without any race conditions.
func MessageBroker(s *Server) {
	log.Println("0 - [SERVER]: Starting MessageBroker")

	// We make a slice to store all of our active connections. Order does not matter.
	s.clients = make([]p.ChatService_ChatStreamServer, 0)

	// We make a channel to receive data from the active goroutines, which listen for incoming traffic from clients.
	s.brokerChan = make(chan BrokerMessage)
	var nextMsg BrokerMessage
	var resp p.Message
	var logicalTime int64 = 0

	for {
		nextMsg = <-s.brokerChan
		switch nextMsg.MsgType {

		// A new client has joined the chat
		case 0:
			// Add the new clients stream pointer to our slice, for easy broadcasting
			s.clients = append(s.clients, *nextMsg.Stream)
			for _, client := range s.clients {
				if client != nil {
					// Every notification delivered to a client is a separate event observed
					logicalTime++
					resp = p.Message{MsgType: 0, Username: nextMsg.User, Text: "", Timestamp: logicalTime}
					log.Printf("%v - [SERVER]: Sending 'Join' message to client %v\n", logicalTime, client)
					_ = client.Send(&resp)
				}
			}
			break

		// Client sends a message
		case 1:
			for _, client := range s.clients {
				if client != nil {
					// Every message delivered to a client is a separate event observed.
					// This also makes it easy to track clients, as their logical time will match the servers.
					logicalTime++
					resp = p.Message{MsgType: 1, Username: nextMsg.User, Text: nextMsg.Text, Timestamp: logicalTime}
					log.Printf("%v - [SERVER]: Sending 'Text' message to client %v\n", logicalTime, client)
					_ = client.Send(&resp)
				}
			}
			break
		// A client has disconnected from the server
		case 2:
			// If not the disconnected sender, broadcast leave msg. If sender, remove from list of active connections.
			for i, client := range s.clients {
				if client != nil {
					logicalTime++
					if client == *nextMsg.Stream {
						// Perform clean-up by removing disconnected stream pointers from slice.
						s.clients = append(s.clients[:i], s.clients[i+1:]...)
						log.Printf("%v - [SERVER]: Removed %v from active connections\n", logicalTime, client)
					} else {
						resp = p.Message{MsgType: 2, Username: nextMsg.User, Text: "", Timestamp: logicalTime}
						log.Printf("%v - [SERVER]: Sending 'Leave' message to client %v\n", logicalTime, client)
						_ = client.Send(&resp)
					}
				}
			}
			break
		}
	}
}

// ChatStream is run as a Goroutine whenever a new connection to the grpc server is made.
// It acts as a per-stream listener, and sends the content it receives to the MessageBroker.
func (s *Server) ChatStream(stream p.ChatService_ChatStreamServer) error {
	var lun string // local username for stream
	for {
		msg, err := stream.Recv()
		if err != nil {
			s.brokerChan <- BrokerMessage{MsgType: 2, User: lun, Text: "", Stream: &stream}
			return err
		}
		switch msg.MsgType {
		case 0:
			lun = msg.Username
			s.brokerChan <- BrokerMessage{MsgType: 0, User: lun, Text: "", Stream: &stream}
			break
		case 1:
			s.brokerChan <- BrokerMessage{MsgType: 1, User: lun, Text: msg.Text, Stream: &stream}
			break
		}
	}
}

func main() {
	s := &Server{}
	go MessageBroker(s)
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("[SERVER]: Server has started, listening at %v", lis.Addr())
	p.RegisterChatServiceServer(grpcServer, s)
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Println("[SERVER]: Server has stopped.")
}
