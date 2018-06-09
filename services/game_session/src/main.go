package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/barreyo/kooky-quiz/pb"
	"github.com/gomodule/redigo/redis"
	"github.com/twinj/uuid"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type server struct {
	gameCodeSize int
	dbPool       *redis.Pool
	hostName     string
}

var (
	crt = "/certs/tls.crt"
	key = "/certs/tls.key"
)

func genClientID() uuid.UUID {
	return uuid.NewV4()
}

func (s server) formatWSAddr(gameName, code string, client uuid.UUID) string {
	return fmt.Sprintf("wss://%s/ws/%s/%s/%s", s.hostName, gameName, code, client)
}

func (s server) New(context context.Context, req *pb.NewSessionRequest) (*pb.NewSessionResponse, error) {

	log.Printf("New Game request for game: %s", req.GameName)

	// Find out if this is a game that exist and we can start a session for it
	// TODO: This should be recorded and fetched from some persistant storage
	// 		 Maybe let games register with DNS and stuff so that the session
	//		 handler can route requests.
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
			log.Printf("Redis error doing EXISTS call: %s", err)
			continue
		}

		if exists {
			log.Printf("Game Code existed already, trying again. (%d / 100)", i)
			continue
		} else {
			break
		}
	}

	wsAddr := s.formatWSAddr(req.GameName, gameCode.code, clientID)
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
	c.Send("SET", gameCode.code, buffer.String())
	c.Send("EXPIRE", gameCode.code, 1*60*60)
	c.Do("EXEC")

	return &pb.NewSessionResponse{
		Code:     gameCode.code,
		ClientId: fmt.Sprintf("%s", clientID),
		WsAddr:   wsAddr}, nil
}

func (s server) Join(context context.Context, req *pb.JoinGameRequest) (*pb.JoinGameResponse, error) {

	log.Printf("Join game request to game: %s, for user %s", req.JoinCode, req.Name)

	playerName := strings.TrimSpace(req.Name)

	if utf8.RuneCountInString(playerName) > 20 {
		return nil, fmt.Errorf("Name too long, only 20 characters allowed")
	}

	if playerName == "" {
		return nil, fmt.Errorf("Name cannot be empty")
	}

	// Get hold of a DB connection from the pool
	c := s.dbPool.Get()
	defer c.Close()

	exists, err := redis.Bool(c.Do("EXISTS", req.JoinCode))
	if err != nil {
		log.Printf("DB Connection error: %s", err)
		return nil, fmt.Errorf("DB Connection error")
	}

	if !exists {
		return nil, fmt.Errorf("No game with code %s exist", req.JoinCode)
	}

	gameStateStr, err := redis.String(c.Do("GET", req.JoinCode))
	if err != nil {
		log.Printf("DB Connection error: %s", err)
		return nil, fmt.Errorf("DB Connection error")
	}

	gameState := pb.GameSession{}
	err = json.Unmarshal([]byte(gameStateStr), &gameState)
	if err != nil {
		log.Printf("Failed to get state from DB: %s", err)
		return nil, fmt.Errorf("DB Connection error")
	}

	for _, p := range gameState.GetPlayers() {
		if p.Name == playerName {
			return nil, fmt.Errorf("The name '%s' is already in use", playerName)
		}
	}

	clientID := genClientID()
	newPlayer := &pb.Player{
		Name:   playerName,
		UserId: fmt.Sprintf("%s", clientID),
		WsAddr: s.formatWSAddr(gameState.GameType, gameState.GameId, clientID),
	}
	gameState.Players = append(gameState.Players, newPlayer)

	stateMarshalled, err := json.Marshal(gameState)
	if err != nil {
		fmt.Printf("Marshal error for new game state: %s", err)
		return nil, fmt.Errorf("Failed to update game state with new player")
	}

	var buffer bytes.Buffer
	buffer.Write(stateMarshalled)

	// TODO: Check errors here as well, might have lost connetion to DB etc
	// 		 do some retries
	c.Send("SET", req.JoinCode, buffer.String())
	c.Send("EXPIRE", req.JoinCode, 5*60*60) // Keep game around for 5h if past lobby state
	c.Do("EXEC")

	return &pb.JoinGameResponse{
		WsAddr: newPlayer.WsAddr,
		UserId: newPlayer.UserId,
	}, nil
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

	hostName := os.Getenv("KOOKY_HOSTNAME")

	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen on %s: %v", addr, err)
	}
	log.Printf("GRPC service listening on %s", addr)

	// Create the TLS credentials
	creds, err := credentials.NewServerTLSFromFile(crt, key)
	if err != nil {
		log.Printf("Could not load TLS keys: %s", err)
	}

	redisAddr := fmt.Sprintf("%s:%d", redisName, redisPort)
	log.Printf("Creating a Redis DB pool with %s", redisAddr)

	dbPool := newDBPool(redisAddr, redisPassword)

	s := grpc.NewServer(grpc.Creds(creds))
	grpcClient := &server{gameCodeSize, dbPool, hostName}
	pb.RegisterGameSessionServiceServer(s, grpcClient)

	log.Printf("Starting API server on port 443")
	api := InitAPIServer(grpcClient)
	go api.Run(":443")

	reflection.Register(s)
	log.Println("Service Reflection registered")
	log.Println("Serving Session service...")

	if err := s.Serve(lis); err != nil {
		log.Printf("Failed to serve %v", err)
	}
}
