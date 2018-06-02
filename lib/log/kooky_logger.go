package kookylog

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func init() {
	env := os.Getenv("ENV")

	if env == "prod" {
		log.SetFormatter(&log.JSONFormatter{})
	}
}
