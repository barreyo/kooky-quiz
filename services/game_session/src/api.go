package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

type apiServer struct{}

func (s apiServer) newGameHandler(c *gin.Context) {
	log.Printf("Got a new game request")
}

func (s apiServer) joinGameHandler(c *gin.Context) {
	log.Printf("Got a join request")
}

/// InitAPIServer config a API server for the session service.
func InitAPIServer() *gin.Engine {
	r := gin.Default()
	server := apiServer{}

	v1 := r.Group("/api/v1/session")
	{
		v1.POST("/new", server.newGameHandler)
		v1.POST("/join", server.joinGameHandler)
	}

	return r
}
