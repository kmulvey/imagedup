package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		fmt.Println(prefix, err)
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}
