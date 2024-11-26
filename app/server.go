package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var fileRoot string

type Path struct {
	FullPath       string
	PathParameters []string
}

type Request struct {
	Method  string
	Path    Path
	Headers map[string]string
	Body    []byte
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
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
	_, err := conn.Read(requestStr)

	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		writeErrorResponse(conn)
		return
	}

	request := parseRequest(requestStr)

	if request.Path.PathParameters[0] == "echo" {
		handleEchoRequest(conn, request)
	} else if request.Path.PathParameters[0] == "user-agent" {
		useragent := request.Headers["User-Agent"]
		writeOKResponse(conn, []byte(useragent))
	} else if request.Path.PathParameters[0] == "files" {
		if request.Method == "GET" {
			writeFileResponse(conn, request.Path.PathParameters[1:])
		} else if request.Method == "POST" {
			createFile(conn, request)
		}
	} else if request.Path.FullPath == "/" {
		writeOKResponse(conn, []byte("HTTP/1.1 200 OK\r\n"))
	} else {
		writeNotFoundResponse(conn)
	}
}

func handleEchoRequest(conn net.Conn, request Request) {
	if len(request.Path.PathParameters) != 2 {
		writeErrorResponse(conn)
		return
	}
	body := []byte(request.Path.PathParameters[1])
	headers := make(map[string]string)
	encodings := strings.Split(request.Headers["Accept-Encoding"], ", ")
	for _, encoding := range encodings {
		if encoding == "gzip" {
			headers["Content-Encoding"] = "gzip"
			body = gzipCompress(body)
		}
	}
	headers["Content-Type"] = "text/plain"
	headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	response := Response{
		Status:  200,
		Headers: headers,
		Body:    body,
	}
	writeResponse(conn, response)
}

func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func writeOKResponse(conn net.Conn, body []byte) {
	headers := make(map[string]string)
	headers["Content-Type"] = "text/plain"
	headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	response := Response{
		Status:  200,
		Headers: headers,
		Body:    body,
	}
	writeResponse(conn, response)
}

func writeNotFoundResponse(conn net.Conn) {
	response := Response{
		Status:  404,
		Headers: make(map[string]string),
		Body:    []byte("404 Not Found"),
	}
	writeResponse(conn, response)
}

func writeErrorResponse(conn net.Conn) {
	response := Response{
		Status:  400,
		Headers: make(map[string]string),
		Body:    []byte("Bad Request"),
	}
	writeResponse(conn, response)
}

func writeServerErrorResponse(conn net.Conn) {
	response := Response{
		Status:  500,
		Headers: make(map[string]string),
		Body:    []byte("Internal Server Error"),
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
	_, err := conn.Write([]byte(fmt.Sprintf("\r\n%s", response.Body)))
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
	}
}

func writeStatusLine(conn net.Conn, statusCode int) {
	_, err := conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %s\r\n", lineForStatusCode(statusCode))))
	if err != nil {
		fmt.Println("Error writing status line: ", err.Error())
	}
}

func writeFileResponse(conn net.Conn, params []string) {
	filename := params[0]
	filePath := fmt.Sprintf("%s/%s", fileRoot, filename)
	content, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		writeNotFoundResponse(conn)
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/octet-stream"
	headers["Content-Length"] = fmt.Sprintf("%d", len(content))
	response := Response{
		Status:  200,
		Headers: headers,
		Body:    content,
	}
	writeResponse(conn, response)
}

func createFile(conn net.Conn, request Request) {
	filePath := fmt.Sprintf("%s/%s", fileRoot, request.Path.PathParameters[1])
	fmt.Printf("Writing %s to file %s\n", request.Body, filePath)
	err := os.WriteFile(filePath, request.Body, 0666)
	if err == nil {
		response := Response{
			Status:  201,
			Headers: make(map[string]string),
			Body:    []byte("201 Created"),
		}
		writeResponse(conn, response)
	} else {
		writeServerErrorResponse(conn)
		log.Fatal(err)
	}
}

func lineForStatusCode(statusCode int) string {
	switch statusCode {
	case 200:
		return "200 OK"
	case 201:
		return "201 Created"
	case 400:
		return "400 Bad Request"
	case 404:
		return "404 Not Found"
	default:
		return "500 Internal Server Error"
	}
}

func writeHeader(conn net.Conn, header string, value string) {
	_, err := conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", header, value)))
	if err != nil {
	}
}

func parseRequest(requestString []byte) Request {
	requestLines := strings.Split(string(requestString), "\r\n")
	requestLine := requestLines[0]
	requestLineParts := strings.Split(requestLine, " ")

	request := Request{
		Method: requestLineParts[0],
		Path:   parsePath(requestLineParts[1]),
	}

	if len(requestLines) > 1 {
		request.Headers = parseHeaders(requestLines[1:])
	}

	contentLength, err := strconv.Atoi(request.Headers["Content-Length"])
	if err == nil {
		request.Body = []byte(requestLines[len(requestLines)-1])[:contentLength]
	}

	return request
}

func parsePath(path string) Path {
	pathParts := strings.Split(path, "/")
	return Path{
		FullPath:       path,
		PathParameters: pathParts[1:],
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
