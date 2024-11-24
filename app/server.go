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
	Command string
	Headers map[string]string
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	requestStr := make([]byte, 1024)
	conn.Read(requestStr)

	request := parseRequest(requestStr)

	if request.Command == "echo" {
		writeOKResponse(conn, request.Path.Parts[1])
	} else if request.Command == "user-agent" {
		useragent := request.Headers["User-Agent"]
		writeOKResponse(conn, useragent)
	} else if request.Path.FullPath == "/" {
		writeOKResponse(conn, "HTTP/1.1 200 OK\r\n")
	} else {
		writeResponse(conn, 404, "Not Found")
	}
}

func writeOKResponse(conn net.Conn, response string) {
	writeResponse(conn, 200, response)
}

func writeResponse(conn net.Conn, status int, response string) {
	writeStatusLine(conn, status)
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
	request.Command = request.Path.Parts[0]

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
