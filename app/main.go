package main

import (
	"fmt"
	"log"
	"net"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatal("Could not create listener")
	}
	defer l.Close()

	for {
		con, err := l.Accept()
		if err != nil {
			return
		}

		go func(con net.Conn) {
			defer con.Close()

			respStr := "HTTP/1.1 200 OK\r\n\r\n"
			resp := []byte(respStr)
			n, err := con.Write(resp)
			if err != nil {
				log.Fatal("Could not reply")
			}

			fmt.Printf("Wrote %v bytes\n", n)
		}(con)
	}
}
