package main

import (
	"log"
	"net"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatal("Could not create listener")
	}
	_, err = l.Accept()

	if err != nil {
		log.Fatal("Error Accepting Connection Request on listener")
	}
}
