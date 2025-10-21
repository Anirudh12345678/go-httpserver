package main

import (
	"fmt"
	"log"
	"net"
	"strings"
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

			buff := make([]byte, 108)
			_, err := con.Read(buff)

			if err != nil {
				log.Fatal("Could not read from connection")
			}
			arr := strings.Split(string(buff), "\n")

			reqLine := arr[0]

			reqUrl := strings.Split(reqLine, " ")[1]

			fmt.Println(reqUrl)
			respStr := "HTTP/1.1 200 OK\r\n\r\n"
			resp := []byte(respStr)

			if reqUrl == "/" {
				n, err := con.Write(resp)

				if err != nil {
					log.Fatal("Could not reply")
				}
				fmt.Printf("Wrote %v bytes\n", n)
			} else {
				con.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			}

		}(con)
	}
}
