package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/twinj/uuid"
)

func genClientID() uuid.UUID {
	return uuid.NewV4()
}

func newGameHandler(c *gin.Context) {
	clientID := genClientID()
	log.Printf("Got a new game request: %s", clientID)
}

func joinGameHandler(c *gin.Context) {
	log.Printf("Got a join request")
}

func InitAPIServer() *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1/game_session")
	{
		v1.POST("/new_game", newGameHandler)
		v1.POST("/join_game", joinGameHandler)
	}

	return r
}
