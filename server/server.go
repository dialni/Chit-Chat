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
func MessageBroker(s *Server) {
	log.Println("[SERVER]: Starting MessageBroker")
	s.clients = make([]p.ChatService_ChatStreamServer, 0)
	s.brokerChan = make(chan BrokerMessage)
	var nextMsg BrokerMessage
	var resp p.Message
	var logicalTime int64 = 0

	for {
		nextMsg = <-s.brokerChan
		logicalTime++
		switch nextMsg.MsgType {
		case 0:
			s.clients = append(s.clients, *nextMsg.Stream)
			resp = p.Message{MsgType: 0, Username: nextMsg.User, Text: "", Timestamp: logicalTime}
			for _, client := range s.clients {
				if client != nil {
					log.Printf("[SERVER]: Sending 'Join' message to client %v\n", client)
					_ = client.Send(&resp)
				}
			}
			break

		case 1:
			resp = p.Message{MsgType: 1, Username: nextMsg.User, Text: nextMsg.Text, Timestamp: logicalTime}
			for _, client := range s.clients {
				if client != nil {
					log.Printf("[SERVER]: Sending 'Text' message to client %v\n", client)
					_ = client.Send(&resp)
				}
			}
			break

		case 2:
			// if not sender, broadcast leave msg. if sender, remove from list
			resp = p.Message{MsgType: 2, Username: nextMsg.User, Text: "", Timestamp: logicalTime}
			for i, client := range s.clients {
				if client != nil {
					if client == *nextMsg.Stream {
						s.clients = append(s.clients[:i], s.clients[i+1:]...)
						log.Printf("[SERVER]: Removed %v from active connections\n", client)
					} else {
						log.Printf("[SERVER]: Sending 'Leave' message to client %v\n", client)
						_ = client.Send(&resp)
					}
				}
			}
			break
		}
	}
}

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

	log.Printf("server listening at %v", lis.Addr())
	p.RegisterChatServiceServer(grpcServer, s)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
