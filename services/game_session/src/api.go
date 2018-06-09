package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/barreyo/kooky-quiz/lib/cors"
	"github.com/barreyo/kooky-quiz/pb"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

type apiServer struct {
	rpcServer *server
	context   context.Context
	wsHub     *Hub
}

func (s apiServer) newGameHandler(c *gin.Context) {
	log.Printf("Got a new game request through the API")

	var req pb.NewSessionRequest
	c.BindJSON(&req)
	res, err := s.rpcServer.New(s.context, &req)

	if err != nil {
		// TODO: Do this error wrapping in a middleware
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": err.Error(), "code": 400}})
		return
	}

	// TODO: Do this wrapping in a middleware
	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (s apiServer) joinGameHandler(c *gin.Context) {
	log.Printf("Got a join request through the API")

	var req pb.JoinGameRequest
	c.BindJSON(&req)
	res, err := s.rpcServer.Join(s.context, &req)

	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": err.Error(), "code": 400}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (s apiServer) wsHandler(c *gin.Context) {

	// Get hold of a DB connection from the pool
	dbConn := s.rpcServer.dbPool.Get()
	defer dbConn.Close()

	gameType := c.Param("game")
	gameID := c.Param("gameId")
	userID := c.Param("userId")

	log.Printf("Request p: %s, %s, %s", gameType, gameID, userID)

	// Check if this is a valid websocket path
	// i.e. the user has done the /join dance and been given a valid adress
	gameStateStr, err := redis.String(dbConn.Do("GET", gameID))
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": "Invalid websocket path, game code does not exist", "code": 400}})
		return
	}

	gameState := pb.GameSession{}
	err = json.Unmarshal([]byte(gameStateStr), &gameState)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": "Invalid websocket path, game code does not exist", "code": 400}})
		return
	}

	userInGame := false
	for _, p := range gameState.GetPlayers() {
		if p.UserId == userID {
			userInGame = true
		}
	}

	if !userInGame {
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": "User is not authorized to join this game", "code": 400}})
		return
	}

	// Do an upgrade and establish connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": gin.H{"message": "Failed to upgrade websocket connection", "code": 400}})
		return
	}

	client := &Client{hub: s.wsHub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.readPump()
	go client.writePump()

	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// InitAPIServer configs an API server for the session service.
func InitAPIServer(rpcServer *server) *gin.Engine {
	r := gin.Default()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run the websocket hub in a goroutine
	hub := newHub()
	go hub.run()

	server := apiServer{rpcServer, ctx, hub}

	// API Endpoints
	v1 := r.Group("/api/v1/session")
	{
		v1.POST("/new", server.newGameHandler)
		v1.POST("/join", server.joinGameHandler)
	}

	// Websocket upgrader endpoints
	ws := r.Group("/ws")
	{
		// Do the WebSocket upgrade and open a connection
		ws.GET("/:game/:gameId/:userId", server.wsHandler)
	}

	cors.SetCORS(r)

	return r
}
