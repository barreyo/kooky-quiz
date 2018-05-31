package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/barreyo/kooky-quiz/pb"
	"github.com/garyburd/redigo/redis"
	"github.com/twinj/uuid"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type server struct {
	gameCodeSize int
	dbPool       *redis.Pool
}

var (
	crt = "/certs/tls.crt"
	key = "/certs/tls.key"
)

func genClientID() uuid.UUID {
	return uuid.NewV4()
}

func (s server) New(context context.Context, req *pb.NewSessionRequest) (*pb.NewSessionResponse, error) {

	log.Printf("New Game request for game: %s", req.GameName)

	// Find out if this is a game that exist and we can start a session for it
	// TODO: This should be recorded in a database or something
	if req.GameName != "kooky-quiz" {
		return nil, fmt.Errorf("%s is not a valid game type", req.GameName)
	}

	clientID := genClientID()
	gameCode := genGameCode(s.gameCodeSize)

	// Get hold of a DB connection from the pool
	c := s.dbPool.Get()
	defer c.Close()

	// Check if the game code is in use
	for i := 0; i <= 100; i++ {

		if i >= 100 {
			return nil, fmt.Errorf("Failed to generate game code")
		}

		exists, err := redis.Bool(c.Do("EXISTS", gameCode.code))
		if err != nil {
			log.Fatalf("Redis error doing EXISTS call: %s", err)
			continue
		}

		if exists {
			log.Printf("Game Code existed already, trying again. (%d / 100)", i)
			continue
		} else {
			break
		}
	}

	// TODO: No hardcoding of the adress
	wsAddr := fmt.Sprintf("ws://dev.kooky.app/ws/%s/%s/%s", req.GameName, gameCode.code, clientID)
	var players []*pb.Player

	gameSession := &pb.GameSession{
		GameId:    gameCode.code,
		GameType:  req.GameName,
		GameState: pb.GameState_LOBBY,
		Players:   players,
		Master:    &pb.Master{ClientId: fmt.Sprintf("%s", clientID), WsAddr: wsAddr},
	}
	gameSessionJSON, err := json.Marshal(gameSession)

	if err != nil {
		return nil, fmt.Errorf("Failed to create the game state. %s", err)
	}

	var buffer bytes.Buffer
	buffer.Write(gameSessionJSON)

	// TODO: Check errors here as well
	c.Do("SET", gameCode.code, buffer.String())

	return &pb.NewSessionResponse{
		Code:     gameCode.code,
		ClientId: fmt.Sprintf("%s", clientID),
		WsAddr:   wsAddr}, nil
}

func (s server) Join(context context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {

	log.Printf("Join game request to game: %s, for user %s", req.JoinCode, req.Name)

	// Get hold of a DB connection from the pool
	c := s.dbPool.Get()
	defer c.Close()

	exists, err := redis.Bool(c.Do("EXISTS", req.JoinCode))
	if err != nil {
		log.Fatalf("DB Connection error: %s", err)
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("No game with code %s exist", req.JoinCode)
	}

	gameStateStr, err := redis.String(c.Do("GET", req.JoinCode))

	if err != nil {
		log.Fatalf("DB Connection error: %s", err)
		return nil, err
	}

	gameState := pb.GameSession{}
	err = json.Unmarshal([]byte(gameStateStr), &gameState)

	if err != nil {
		log.Fatalf("Failed to get state from DB: %s", err)
		return nil, err
	}

	return &pb.JoinGameResponse{}, nil
}

func newDBPool(addr string, pass string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000, // max number of connections
		Dial: func() (redis.Conn, error) {

			// TODO: Dial with timeout
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				panic(err.Error())
			}
			// _, err = c.Do("AUTH", pass)
			// if err != nil {
			//	panic(err.Error())
			// }
			return c, err
		},
	}
}

func main() {
	var port, gameCodeSize, redisPort int
	var redisName, redisPassword string
	flag.IntVar(&port, "port", 50051, "listen port for the service")
	flag.IntVar(&gameCodeSize, "game-code-size", 5, "length of game codes being used")
	flag.IntVar(&redisPort, "redis-port", 6379, "redis connection port")
	flag.StringVar(&redisName, "redis-name", "redis-master", "dns for redis service")
	flag.StringVar(&redisPassword, "redis-pass", "test", "auth for redis")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", "localhost", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	log.Printf("GRPC service listening on %s", addr)

	// Create the TLS credentials
	creds, err := credentials.NewServerTLSFromFile(crt, key)
	if err != nil {
		log.Fatalf("Could not load TLS keys: %s", err)
	}

	redisAddr := fmt.Sprintf("%s:%d", redisName, redisPort)
	log.Printf("Creating a Redis DB pool with %s", redisAddr)

	dbPool := newDBPool(redisAddr, redisPassword)

	s := grpc.NewServer(grpc.Creds(creds))
	grpcClient := &server{gameCodeSize, dbPool}
	pb.RegisterGameSessionServiceServer(s, grpcClient)

	log.Printf("Starting API server on port 443")
	api := InitAPIServer(grpcClient)
	go api.Run(":443")

	reflection.Register(s)
	log.Println("Service Reflection registered")
	log.Println("Serving Session service...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve %v", err)
	}
}
