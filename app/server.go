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
	Headers map[string]string
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
	} else if request.Path.Parts[0] == "user-agent" {
		useragent := request.Headers["User-Agent"]
		writeResponse(conn, useragent)
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

func parseRequest(requestStrings []byte) Request {
	requestLines := strings.Split(string(requestStrings), "\r\n")
	requestLine := requestLines[0]
	requestLineParts := strings.Split(requestLine, " ")

	request := Request{
		Method: requestLineParts[0],
		Path:   parsePath(requestLineParts[1]),
	}

	if len(requestLines) > 1 {
		request.Headers = parseHeaders(requestLines[1:])
	}

	return request
}

func parsePath(path string) Path {
	pathParts := strings.Split(path, "/")
	return Path{
		FullPath: path,
		Parts:    pathParts[1:],
	}
}

func parseHeaders(headerStrings []string) map[string]string {
	headers := make(map[string]string)
	for _, headerString := range headerStrings {
		headerParts := strings.Split(headerString, ": ")
		if len(headerParts) == 2 {
			headers[headerParts[0]] = headerParts[1]
		}
	}
	return headers
}
