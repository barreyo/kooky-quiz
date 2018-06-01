package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/barreyo/kooky-quiz/lib/cors"
	"github.com/barreyo/kooky-quiz/pb"
	"github.com/gin-gonic/gin"
)

type apiServer struct {
	rpcServer *server
	context   context.Context
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

// InitAPIServer configs an API server for the session service.
func InitAPIServer(rpcServer *server) *gin.Engine {
	r := gin.Default()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := apiServer{rpcServer, ctx}

	v1 := r.Group("/api/v1/session")
	{
		v1.POST("/new", server.newGameHandler)
		v1.POST("/join", server.joinGameHandler)
	}
	cors.SetCORS(r)

	return r
}
