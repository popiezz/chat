package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

var (
	Port     = "8008"
	SafeMode = false
)

var (
	broadcast = make(chan BroadcastMessage)
	clients   = make(map[net.Conn]string)
	mu        sync.Mutex
)

type BroadcastMessage struct {
	message  string
	sender   net.Conn
	username string
}

func broadcaster() {
	for {
		msg := <-broadcast
		mu.Lock()
		for conn := range clients {
			if conn != msg.sender {
				_, err := conn.Write([]byte(msg.message))
				if err != nil {
					log.Printf("ERROR broadcasting to %s: %s\n", SafeRemoteAddress(conn), err)
				}
			}
		}
		mu.Unlock()
	}
}

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
		log.Printf("ERROR showing welcome message: %s\n", err)
	}
	log.Printf("Accepted connection from %s\n", SafeRemoteAddress(conn))
}

func getUsername(conn net.Conn) string {
	tutorial := []byte("Type your username: ")
	_, err := conn.Write(tutorial)
	if err != nil {
		log.Printf("ERROR sending username prompt: %s\n", err)
		return fmt.Sprintf("User%d", len(clients)+1) // Fallback username
	}
	reader := bufio.NewReader(conn)
	user, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("ERROR reading username: %s\n", err)
		return fmt.Sprintf("User%d", len(clients)+1) // Fallback username
	}
	user = strings.TrimSpace(user)
	if user == "" {
		user = fmt.Sprintf("User%d", len(clients)+1)
	}
	msgStart := []byte("You can now start typing! Enjoy!\n")
	_, err = conn.Write(msgStart)
	if err != nil {
		log.Printf("ERROR sending start message: %s\n", err)
	}
	return user
}

func acceptMessage(conn net.Conn, user string) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		cleanMsg := strings.TrimSpace(msg)
		if strings.ToUpper(cleanMsg) == "BYE" {
			conn.Write([]byte("Goodbye\n"))
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			// Broadcast disconnection
			broadcast <- BroadcastMessage{
				message:  fmt.Sprintf("%s has disconnected\n", user),
				sender:   conn,
				username: user,
			}
			conn.Close()
			return
		}
		// Send message with username
		broadcast <- BroadcastMessage{
			message:  fmt.Sprintf("%s: %s\n", user, cleanMsg),
			sender:   conn,
			username: user,
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("ERROR reading message from %s: %s\n", user, err)
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
		broadcast <- BroadcastMessage{
			message:  fmt.Sprintf("%s has disconnected\n", user),
			sender:   conn,
			username: user,
		}
		conn.Close()
	}
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	message := []byte("\n       --- Welcome--- \n If you'd like to exit, please type 'BYE'\n")
	showWelcome(message, conn)
	user := getUsername(conn)
	if user == "" {
		return // Connection closed or invalid username
	}
	mu.Lock()
	clients[conn] = user
	// Broadcast new connection
	for otherConn := range clients {
		if otherConn != conn {
			_, err := otherConn.Write([]byte(fmt.Sprintf("%s has connected\n", user)))
			if err != nil {
				log.Printf("ERROR notifying %s of new connection: %s\n", SafeRemoteAddress(otherConn), err)
			}
		}
	}
	// Send list of existing clients to new client
	for otherConn, otherUser := range clients {
		if otherConn != conn {
			_, err := conn.Write([]byte(fmt.Sprintf("%s is available to chat\n", otherUser)))
			if err != nil {
				log.Printf("ERROR sending client list to %s: %s\n", user, err)
				mu.Unlock()
				return
			}
		}
	}
	mu.Unlock()
	log.Printf("%s joined as %s\n", conn.RemoteAddr().String(), user)
	acceptMessage(conn, user)
}

func main() {
	ln, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Printf("ERROR starting server: %s\n", err)
		return
	}
	fmt.Printf("Currently accepting connections on port %s\n", Port)
	go broadcaster() // Start broadcaster once
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("ERROR accepting connection: %s\n", err)
			continue // Continue accepting other connections
		}
		go HandleConnection(conn)
	}
}
