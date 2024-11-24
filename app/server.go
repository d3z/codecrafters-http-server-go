package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

var fileRoot string

type Path struct {
	FullPath string
	PathParameters    []string
}

type Request struct {
	Method string
	Path   Path
	Headers map[string]string
}

type Response struct {
	Status int
	Headers map[string]string
	Body []byte
}

func main() {
	flag.StringVar(&fileRoot, "directory", ".", "The directory to serve files from")
	flag.Parse()

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

	if request.Path.PathParameters[0] == "echo" {
		writeOKResponse(conn, []byte(request.Path.PathParameters[1]))
	} else if request.Path.PathParameters[0] == "user-agent" {
		useragent := request.Headers["User-Agent"]
		writeOKResponse(conn, []byte(useragent))
	} else if request.Path.PathParameters[0] == "files" {
		writeFileResponse(conn, request.Path.PathParameters)
	} else if request.Path.FullPath == "/" {
		writeOKResponse(conn, []byte("HTTP/1.1 200 OK\r\n"))
	} else {
		writeNotFoundResponse(conn)
	}
}

func writeOKResponse(conn net.Conn, body []byte) {
	headers := make(map[string]string)
	headers["Content-Type"] = "text/plain"
	headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	response := Response {
		Status: 200,
		Headers: headers,
		Body: body,
	}
	writeResponse(conn, response)
}

func writeNotFoundResponse(conn net.Conn) {
	response := Response {
		Status: 404,
		Headers: make(map[string]string),
		Body: []byte("404 Not Found"),
	}
	writeResponse(conn, response)
}

func writeErrorResponse(conn net.Conn) {
	response := Response {
		Status: 400,
		Headers: make(map[string]string),
		Body: []byte("Bad request"),
	}
	writeResponse(conn, response)
}

func writeResponse(conn net.Conn, response Response) {
	writeStatusLine(conn, response.Status)
	for header, value := range response.Headers {
		writeHeader(conn, header, value)
	}
	if response.Headers["Content-Length"] == "" {
		writeHeader(conn, "Content-Length", fmt.Sprintf("%d", len(response.Body)))
	}
	conn.Write([]byte("\r\n"))
	conn.Write(response.Body)
}

func writeStatusLine(conn net.Conn, statusCode int) {
	conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %s\r\n", lineForStatusCode(statusCode))))
}

func writeFileResponse(conn net.Conn, args []string) {
	if len(args) != 2 {
		writeErrorResponse(conn)
		return
	}
	filename := args[1]
	filePath := fmt.Sprintf("%s/%s", fileRoot, filename)
	content, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		writeNotFoundResponse(conn)
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/octet-stream"
	headers["Content-Length"] = fmt.Sprintf("%d", len(content))
	response := Response{
		Status: 200,
		Headers: headers,
		Body: content,
	}
	writeResponse(conn, response)
}

func lineForStatusCode(statusCode int) string {
	switch statusCode {
	case 200:
		return "200 OK"
	case 400:
		return "400 Bad Request"
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
		PathParameters:    pathParts[1:],
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
