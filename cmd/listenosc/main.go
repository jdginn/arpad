package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hypebeast/go-osc/osc"
)

func main() {
	port := flag.Int("port", 0, "TCP port to listen for OSC messages")
	flag.Parse()

	if *port == 0 {
		fmt.Println("Usage: listenosc -port <port>")
		os.Exit(1)
	}
	addr := "0.0.0.0:" + strconv.Itoa(*port)

	// Create a dispatcher that prints all messages
	dispatcher := osc.NewStandardDispatcher()
	dispatcher.AddMsgHandler("*", func(msg *osc.Message) {
		fmt.Printf("Received OSC message: %s %v\n", msg.Address, msg.Arguments)
	})

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: dispatcher,
		// By default, go-osc only accepts UDP. TCP requires additional setup,
		// but for simple use UDP is standard for OSC.
	}

	fmt.Printf("Listening for OSC messages on %s (UDP)...\n", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start OSC server: %v", err)
	}
}
