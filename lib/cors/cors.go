package cors

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetCORS sets the CORS headers on a gin router based on the KOOKY_HOSTNAME
// environment variable.
func SetCORS(router *gin.Engine) {
	hostName := os.Getenv("KOOKY_HOSTNAME")
	origin := fmt.Sprintf("https://%s", hostName)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{origin},
		AllowMethods:     []string{"POST", "GET"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		AllowAllOrigins:  false,
		MaxAge:           12 * time.Hour,
	}))
}
