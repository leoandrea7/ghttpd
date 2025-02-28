package main

import (
  "bufio"
  "errors"
  "flag"
  "fmt"
  "io"
  "log"
  "mime"
  "net"
  "net/url"
  "os"
  "path/filepath"
  "runtime"
  "strings"
  "time"
)

var (
  port string
  dir  string
  workers int
)

func main() {

  flag.StringVar(&port, "p", "8080", "Server port")
  flag.StringVar(&dir, "d", ".", "Directory to serve")
  flag.IntVar(&workers, "w", runtime.NumCPU(), "Number of workers")
  flag.Parse()

  if _, err := os.Stat(dir); os.IsNotExist(err) {
    log.Fatalf("Error: directory %s not exist\n", dir)
  }

  listener, err := net.Listen("tcp", ":"+port)
  if err != nil {
    log.Fatalf("Error: %v", err)
    return
  }
  defer listener.Close()

  log.Println("Listening on port " + port)

  connChan := make(chan net.Conn)

  for i := range workers {
    go func(workerID int) {
      for conn := range connChan {
          log.Printf("Worker %d: handling connection", workerID)
          handleConnection(conn)
      }
    }(i)
  }
  
  for {

    conn, err := listener.Accept()
    conn.SetDeadline(time.Now().Add(5 * time.Second))
    if err != nil {
      log.Fatalf("Error: %v", err)
      return
    }
    connChan <- conn
  }
}

func handleConnection(conn net.Conn) {

  defer conn.Close()

  method, path, version, err := parseRequest(conn)

  if err != nil {
    log.Printf("Error parsing request: %v", err)
    sendError(conn, 400, "Bad Request")
    return
  }

  log.Printf("New Request [Method: %s, Path: %s, Version: %s]", method, path, version)

  if err := validateRequest(method, version); err != nil {
    sendError(conn, 400, err.Error())
    return
  }
  
  serveResource(conn, path)
}

func serveResource(conn net.Conn, path string) {

  fullPath := filepath.Join(dir, path)
  
  fi, err := os.Stat(fullPath)
  if os.IsNotExist(err) {
    sendError(conn, 404, "Not Found")
    return
  } else if err != nil {
    sendError(conn, 500, "Internal Server Error")
    return
  }

  if fi.IsDir() {
    generateDirectoryListing(conn, path, fullPath)
  } else {
    sendFile(conn, fullPath)
  }
}

func validateRequest(method, version string) error {
  if !strings.HasPrefix(version, "HTTP") {
    return fmt.Errorf("invalid HTTP version")
  }

  if method != "GET" {
    return fmt.Errorf("method not allowed")
  }

  return nil
}

// parseRequest reads the first line from the given connection, parses it, and returns the HTTP method, path, and version.
// If the request is invalid, it returns an error instead.
// HTTP Request e.g.:
// GET /test HTTP/1.1
// Host: www.example.com
// User-Agent: curl/7.64.1
// Accept: */*
//
// username=foo&password=bar
//
func parseRequest(conn net.Conn) (string, string, string, error) {

  firstLine, err := bufio.NewReader(conn).ReadString('\n')
  if err != nil {
    log.Printf("Error: %v", err)
    return "", "", "", errors.New("invalid request format")
  }

  parts := strings.Split(firstLine, " ")
  if len(parts) != 3 {
    log.Printf("Error: Invalid request")
    return "", "", "", fmt.Errorf("invalid Request line")
  }

  method, rawPath, version := parts[0], parts[1], parts[2]

  path, err := url.PathUnescape(rawPath)
  if err != nil {
    return "", "", "", fmt.Errorf("invalid URL encoding")
  }

  return method, path, version, nil
}

func sendFile(conn net.Conn, path string) {
  
  file, err := os.Open(path)

  if err != nil {
    sendError(conn, 500, "Internal Server Error")
    return
  }

  defer file.Close()

  ext := filepath.Ext(path)
  contentType := mime.TypeByExtension(ext)

  if contentType == "" {
    contentType = "application/octet-stream"
  }

  info, err := file.Stat()
  if err != nil {
    sendError(conn, 500, "Internal Server Error")
    return
  }
  
  header := fmt.Sprintf(
    "HTTP/1.1 200 OK\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", contentType, info.Size())
  conn.Write([]byte(header))
  io.Copy(conn, file)
}

func generateDirectoryListing(conn net.Conn, path string, fullPath string) {

  files, err := os.ReadDir(fullPath)
  if err != nil {
    sendError(conn, 500, "Internal Server Error")
    return
  }

  var builder strings.Builder

  builder.WriteString("<html><head><title>Directory Listing</title></head><body><h1>Directory Listing</h1><ul>")
  
  for _, file := range files {
    relativePath := filepath.Join(strings.TrimPrefix(path, "."), file.Name())
    builder.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", relativePath, file.Name()))
  }
  builder.WriteString("</ul></body></html>")
  
  response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n%s", builder.Len(), builder.String())
  conn.Write([]byte(response))
}

func sendError(conn net.Conn, code int, message string) {
  response := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", code, message, len(message), message)
  conn.Write([]byte(response))
}