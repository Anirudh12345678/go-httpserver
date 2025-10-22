package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

var registeredRoutes = []string{"/", "/user-agent"}

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

			buff := make([]byte, 4096)
			_, err := con.Read(buff)

			if err != nil {
				log.Fatal("Could not read from connection")
			}

			handleUrl(&buff, &con)

		}(con)
	}
}

func extractHeaders(buffStr string) (map[string]string, string) {
	parts := strings.SplitN(buffStr, "\r\n\r\n", 2)
	headerPart := parts[0]
	reqBody := parts[1]
	lines := strings.Split(headerPart, "\r\n")
	headers := lines[1:]

	headerMap := make(map[string]string)

	for _, line := range headers {
		parts := strings.SplitN(line, ":", 2)

		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		headerMap[key] = val
	}

	return headerMap, reqBody
}

func handleUrl(conBuff *[]byte, con *net.Conn) {
	var out bool = false
	buffStr := string(*conBuff)
	reqLine := strings.Split(buffStr, "\n")[0]
	reqType := strings.Split(reqLine, " ")[0]
	reqUrl := strings.Split(reqLine, " ")[1]
	fmt.Println(reqUrl)
	headerMap, body := extractHeaders(buffStr)

	if reqType == "POST" {
		handlePost(con, &body, &reqUrl, &out)
	}
	if _, exists := routeSet[reqUrl]; exists {
		switch reqUrl {
		case "/":
			n, err := (*con).Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			out = true
			if err != nil {
				log.Fatal("Could not write response")
			}

			fmt.Printf("Wrote %v bytes\n", n)
		case "/user-agent":
			if value, ok := headerMap["user-agent"]; ok {
				resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(value), value)
				n, err := (*con).Write([]byte(resp))

				if err != nil {
					log.Fatal("Could not Write back")
				}
				fmt.Printf("Wrote %v bytes\n", n)
				out = true
			}

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

	re = regexp.MustCompile(`/files/([^/]+)$`)
	match = re.FindStringSubmatch(reqUrl)

	if len(match) > 1 {
		value := match[1]
		filePath := fmt.Sprintf("app/files/%s", value)
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Println("Error reading file", err)
			resp := "HTTP/1.1 404 Not Found\r\n\r\n"
			(*con).Write([]byte(resp))
			out = true
			return
		}
		content := string(data)
		resp := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(content), content)
		fmt.Println(resp)
		(*con).Write([]byte(resp))
		out = true
	}

	if !out {
		(*con).Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func handlePost(con *net.Conn, body *string, url *string, out *bool) {
	re := regexp.MustCompile(`/files/([^/]+)$`)

	match := re.FindStringSubmatch(*url)
	if len(match) > 1 {
		fileName := match[1]
		filePath := fmt.Sprintf("app/files/%s", fileName)
		file, err := os.Create(filePath)
		if err != nil {
			log.Println("Could not create requested file")
			resp := "HTTP/1.1 404 Not Found\r\n\r\n"
			(*con).Write([]byte(resp))
			*out = true
			return
		}

		_, err = file.Write([]byte(*body))
		if err != nil {
			log.Println("Could not create requested file")
			resp := "HTTP/1.1 404 Not Found\r\n\r\n"
			(*con).Write([]byte(resp))
			*out = true
			return
		}

		(*con).Write([]byte("HTTP/1.1 201 Created\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
		*out = true
	}
}
