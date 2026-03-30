package main

import (
	"log"

	"P4wnP12026/internal/server"
)

func main() {
	srv := server.New()
	log.Printf("EchoPI listening on %s", srv.Addr())
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
