package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

type Request struct {
	Method string
	Path   string
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	requestStr := make([]byte, 1024)
	_, err = conn.Read(requestStr)

	request := parseRequest(requestStr)

	if request.Path != "/" {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
}

func parseRequest(request []byte) Request {
	requestLines := strings.Split(string(request), "\r\n")
	requestLine := requestLines[0]
	requestLineParts := strings.Split(requestLine, " ")
	return Request{
		Method: requestLineParts[0],
		Path:   requestLineParts[1],
	}
}
