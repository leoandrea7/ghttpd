package main

import (
	"bufio"
	"log"
	"fmt"
	"net"
	"os"
)

func main() {

	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		log.Fatalf("Error: %v", err)
		return
	}

	for {
        reader := bufio.NewReader(os.Stdin)
        fmt.Print("Text to send: ")
        text, _ := reader.ReadString('\n')
        fmt.Fprintf(conn, text + "\n")
        message, _ := bufio.NewReader(conn).ReadString('\n')
        fmt.Println("Message from server: "+message)
    }
}