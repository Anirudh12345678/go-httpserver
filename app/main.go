package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

type Request struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    string
}

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       string
}

type HandlerFunc func(req *Request) Response

// --- Router ---

type Router struct {
	staticRoutes  map[string]HandlerFunc
	dynamicRoutes []struct {
		pattern *regexp.Regexp
		handler HandlerFunc
	}
}

func NewRouter() *Router {
	return &Router{
		staticRoutes: make(map[string]HandlerFunc),
		dynamicRoutes: []struct {
			pattern *regexp.Regexp
			handler HandlerFunc
		}{},
	}
}

func (r *Router) Handle(path string, handler HandlerFunc) {
	r.staticRoutes[path] = handler
}

func (r *Router) HandleDynamic(pattern string, handler HandlerFunc) {
	re := regexp.MustCompile(pattern)
	r.dynamicRoutes = append(r.dynamicRoutes, struct {
		pattern *regexp.Regexp
		handler HandlerFunc
	}{re, handler})
}

func (r *Router) FindHandler(path string) HandlerFunc {
	if handler, ok := r.staticRoutes[path]; ok {
		return handler
	}
	for _, route := range r.dynamicRoutes {
		if route.pattern.MatchString(path) {
			return route.handler
		}
	}
	return nil
}

// --- HTTP Parsing ---

func parseRequest(conn net.Conn) (*Request, error) {
	reader := bufio.NewReader(conn)
	reqLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	reqLine = strings.TrimRight(reqLine, "\r\n")

	parts := strings.Fields(reqLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("malformed request line: %v", reqLine)
	}
	method, path, version := parts[0], parts[1], parts[2]

	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		k, v, _ := strings.Cut(line, ":")
		headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	var body string
	if cl, ok := headers["content-length"]; ok {
		length := 0
		fmt.Sscanf(cl, "%d", &length)
		buf := make([]byte, length)
		_, err := reader.Read(buf)
		if err != nil {
			return nil, err
		}
		body = string(buf)
	}

	return &Request{
		Method:  method,
		Path:    path,
		Version: version,
		Headers: headers,
		Body:    body,
	}, nil
}

// --- Response Writer ---

func writeResponse(conn net.Conn, res Response) {
	if res.Headers == nil {
		res.Headers = make(map[string]string)
	}

	if _, ok := res.Headers["Content-Length"]; !ok {
		res.Headers["Content-Length"] = fmt.Sprintf("%d", len(res.Body))
	}
	if _, ok := res.Headers["Connection"]; !ok {
		res.Headers["Connection"] = "close"
	}

	fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\n", res.StatusCode, res.StatusText)
	for k, v := range res.Headers {
		fmt.Fprintf(conn, "%s: %s\r\n", k, v)
	}
	fmt.Fprint(conn, "\r\n")
	fmt.Fprint(conn, res.Body)
}

func rootHandler(req *Request) Response {
	return Response{200, "OK", nil, ""}
}

func userAgentHandler(req *Request) Response {
	ua := req.Headers["user-agent"]
	return Response{200, "OK", map[string]string{"Content-Type": "text/plain"}, ua}
}

func echoHandler(req *Request) Response {
	re := regexp.MustCompile(`^/echo/(.+)$`)
	m := re.FindStringSubmatch(req.Path)
	if len(m) > 1 {
		body := m[1]
		return Response{200, "OK", map[string]string{"Content-Type": "text/plain"}, body}
	}
	return Response{404, "Not Found", nil, ""}
}

func fileHandler(req *Request) Response {
	re := regexp.MustCompile(`^/files/(.+)$`)
	m := re.FindStringSubmatch(req.Path)
	if len(m) < 2 {
		return Response{404, "Not Found", nil, ""}
	}
	filename := m[1]
	filepath := fmt.Sprintf("app/files/%s", filename)

	switch req.Method {
	case "GET":
		data, err := os.ReadFile(filepath)
		if err != nil {
			return Response{404, "Not Found", nil, ""}
		}
		return Response{200, "OK", map[string]string{"Content-Type": "application/octet-stream"}, string(data)}
	case "POST":
		os.WriteFile(filepath, []byte(req.Body), 0644)
		return Response{201, "Created", nil, ""}
	case "DELETE":
		os.Remove(filepath)
		return Response{200, "OK", nil, ""}
	default:
		return Response{405, "Method Not Allowed", nil, ""}
	}
}

// Main Loop...

func main() {
	router := NewRouter()
	router.Handle("/", rootHandler)
	router.Handle("/user-agent", userAgentHandler)
	router.HandleDynamic(`^/echo/[^/]+$`, echoHandler)
	router.HandleDynamic(`^/files/[^/]+$`, fileHandler)

	listener, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server running on port 4221")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			req, err := parseRequest(conn)
			if err != nil {
				log.Println("Failed to parse request:", err)
				return
			}
			handler := router.FindHandler(req.Path)
			if handler == nil {
				writeResponse(conn, Response{404, "Not Found", nil, ""})
				return
			}
			res := handler(req)
			writeResponse(conn, res)
		}(conn)
	}
}

