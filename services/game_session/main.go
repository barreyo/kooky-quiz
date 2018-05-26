package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/barreyo/kooky-quiz/pb"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type server struct{}

func (s server) Get(context context.Context, req *pb.QuestionRequest) (*pb.QuestionResponse, error) {

	log.Printf("Recieved request with id: %d", req.Id)

	id := req.Id
	answers := []*pb.Answer{
		&pb.Answer{Answer: "Black"},
		&pb.Answer{Answer: "Brown"},
		&pb.Answer{Answer: "White"},
		&pb.Answer{Answer: "Gray"},
	}
	question := &pb.Question{
		Question:     fmt.Sprintf("What color is Betsy? %d", id),
		Alternatives: answers,
	}
	return &pb.QuestionResponse{Question: question}, nil
}

var (
	crt = "config/dev-certs/server.crt"
	key = "config/dev-certs/server.key"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 3000, "listen port for the service")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", "localhost", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	log.Printf("Listening on %s", addr)

	// Create the TLS credentials
	creds, err := credentials.NewServerTLSFromFile(crt, key)
	if err != nil {
		log.Fatalf("Could not load TLS keys: %s", err)
	}

	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterQuestionServiceServer(s, &server{})

	reflection.Register(s)
	log.Println("Reflection registered")
	log.Println("Serving Question service...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve %v", err)
	}
}
