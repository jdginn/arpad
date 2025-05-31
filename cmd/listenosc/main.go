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
	port := flag.Int("port", 0, "port to listen for OSC messages")
	flag.Parse()

	if *port == 0 {
		fmt.Println("Usage: listenosc -port <port>")
		os.Exit(1)
	}
	addr := "0.0.0.0:" + strconv.Itoa(*port)
	// addr := "192.168.1.146:" + strconv.Itoa(*port)

	// Create a dispatcher that prints all messages
	dispatcher := osc.NewStandardDispatcher()
	dispatcher.AddMsgHandler("*", func(msg *osc.Message) {
		tt, _ := msg.TypeTags()
		fmt.Printf("Received OSC message: %s %s %v\n", msg.Address, tt, msg.Arguments)
		// bytes, _ := msg.MarshalBinary()
		// fmt.Printf("Received OSC message: %x\n", bytes)
	})

	server := &osc.Server{
		Addr:       addr,
		Dispatcher: dispatcher,
	}

	fmt.Printf("Listening for OSC messages on %s (UDP)...\n", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start OSC server: %v", err)
	}
}
