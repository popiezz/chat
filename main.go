package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

var (
	Port     = "8008"
	SafeMode = false
)

func SafeRemoteAddress(conn net.Conn) string {
	if SafeMode {
		return "[REDACTED]"
	} else {
		return conn.RemoteAddr().String()
	}
}

func showWelcome(msg []byte, conn net.Conn) {
	_, err := conn.Write(msg)
	if err != nil {
		log.Printf("ERROR showing welcome message. %s\n", err)
	}
	log.Printf("Accepted connection from %s\n", SafeRemoteAddress(conn))
}

// function needed to read the message from the client
func acceptMessage(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Client %s disconnected.\n", conn.RemoteAddr().String())
			} else {
				log.Printf("ERROR reading from client message: %s\n", err)
			}
			return
		}

		cleanMsg := strings.TrimSpace(msg)
		upperMsg := strings.ToUpper(cleanMsg)

		if strings.Contains(upperMsg, "BYE") {
			conn.Write([]byte("Goodbye!\n"))
			log.Println("Client has closed the connection.")
			os.Exit(0)
		}
		log.Printf("Client : %s", msg)
	}
}

func HandleConnection(conn net.Conn) {
	message := []byte("\n       --- Welcome to PipChat --- \n If you'd like to exit, please type 'BYE'\n")
	showWelcome(message, conn)
	go acceptMessage(conn)

	// function needed to read the message from the server
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := conn.Write([]byte("Pip: " + text + "\n"))
			if err != nil {
				log.Printf("ERROR sending message to client : %s\n", conn.RemoteAddr().String())
				return
			}
		}
	}()
}

func main() {
	ln, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Printf("ERROR starting server: %s\n", err)
		return
	}
	fmt.Printf("Currently accepting connections on port %s\n", Port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("No connection accepted. ERROR : %s\n", err)
			return
		}
		go HandleConnection(conn)
	}
}
