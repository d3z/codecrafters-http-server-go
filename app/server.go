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

type Path struct {
	FullPath string
	Parts    []string
}

type Request struct {
	Method string
	Path   Path
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

	if request.Path.Parts[0] == "echo" {
		writeResponse(conn, request.Path.Parts[1])
	} else if request.Path.FullPath == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func writeResponse(conn net.Conn, response string) {
	writeStatusLine(conn, 200)
	writeHeader(conn, "Content-Type", "text/plain")
	writeHeader(conn, "Content-Length", fmt.Sprintf("%d", len(response)))
	conn.Write([]byte(fmt.Sprintf("\r\n%s", response)))
}

func writeStatusLine(conn net.Conn, statusCode int) {
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %s\r\n", lineForStatusCode(statusCode))))
}

func lineForStatusCode(statusCode int) string {
	switch statusCode {
	case 200:
		return "200 OK"
	case 404:
		return "404 Not Found"
	default:
		return "500 Internal Server Error"
	}
}

func writeHeader(conn net.Conn, header string, value string) {
	conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", header, value)))
}

func parseRequest(request []byte) Request {
	requestLines := strings.Split(string(request), "\r\n")
	requestLine := requestLines[0]
	requestLineParts := strings.Split(requestLine, " ")
	return Request{
		Method: requestLineParts[0],
		Path:   parsePath(requestLineParts[1]),
	}
}

func parsePath(path string) Path {
	pathParts := strings.Split(path, "/")
	return Path{
		FullPath: path,
		Parts:    pathParts[1:],
	}
}
