package main

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

var registeredRoutes = []string{"/"}

var routeSet = make(map[string]struct{}, len(registeredRoutes))

func main() {
	for _, route := range registeredRoutes {
		routeSet[route] = struct{}{}
	}

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

			handleUrl(&buff, &con)
			// arr := strings.Split(string(buff), "\n")
			//
			// reqLine := arr[0]
			//
			// reqUrl := strings.Split(reqLine, " ")[1]
			//
			// fmt.Println(reqUrl)
			// respStr := "HTTP/1.1 200 OK\r\n\r\n"
			// resp := []byte(respStr)
			//
			// if reqUrl == "/" {
			// 	n, err := con.Write(resp)
			//
			// 	if err != nil {
			// 		log.Fatal("Could not reply")
			// 	}
			// 	fmt.Printf("Wrote %v bytes\n", n)
			// } else {
			// 	con.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			// }

		}(con)
	}
}

func handleUrl(conBuff *[]byte, con *net.Conn) {
	var out bool = false
	buffStr := string(*conBuff)
	reqLine := strings.Split(buffStr, "\n")[0]

	reqUrl := strings.Split(reqLine, " ")[1]

	if _, exists := routeSet[reqUrl]; exists {
		switch reqUrl {
		case "/":
			n, err := (*con).Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			out = true
			if err != nil {
				log.Fatal("Could not write response")
			}

			fmt.Printf("Wrote %v bytes\n", n)
		}
	}

	re := regexp.MustCompile(`^/echo/([^/]+)$`)
	match := re.FindStringSubmatch(reqUrl)

	if len(match) > 1 {
		value := match[1]
		resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(value), value)
		fmt.Println(resp)

		(*con).Write([]byte(resp))
		out = true
	}

	if !out {
		(*con).Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
