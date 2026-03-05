// wsclient is a simple WebSocket client that bridges stdin/stdout with a
// WebSocket server. It reads JSON lines from stdin and sends them to the
// server, and prints all received messages to stdout as JSON lines.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: wsclient ws://host:port/ws/ROOM\n")
		os.Exit(1)
	}
	url := os.Args[1]

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Handle interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Read from server -> stdout
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			// Pretty-print compact JSON
			var parsed interface{}
			if json.Unmarshal(message, &parsed) == nil {
				compact, _ := json.Marshal(parsed)
				fmt.Println(string(compact))
			} else {
				fmt.Println(string(message))
			}
		}
	}()

	// Read from stdin -> server
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				log.Printf("write: %v", err)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-interrupt:
	}
}
