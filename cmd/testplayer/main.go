// testplayer connects to a game server via WebSocket and plays through
// the full game lifecycle autonomously. Used for L4 subagent testing.
//
// Usage: testplayer -url ws://... -nickname Alice -host
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "", "WebSocket URL")
	nickname := flag.String("nickname", "Player", "Player nickname")
	isHost := flag.Bool("host", false, "Is this player the host?")
	numPlayers := flag.Int("players", 2, "Expected number of players (host waits for this many)")
	flag.Parse()

	if *url == "" {
		log.Fatal("--url is required")
	}

	logf := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, "[%s] %s\n", *nickname, fmt.Sprintf(format, args...))
	}

	conn, _, err := websocket.DefaultDialer.Dial(*url, nil)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Message channel
	msgs := make(chan map[string]interface{}, 128)
	go func() {
		defer close(msgs)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]interface{}
			if json.Unmarshal(data, &m) == nil {
				msgs <- m
			}
		}
	}()

	send := func(v interface{}) {
		data, _ := json.Marshal(v)
		logf("SEND: %s", data)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	waitFor := func(msgType string, timeout time.Duration) map[string]interface{} {
		deadline := time.After(timeout)
		for {
			select {
			case m, ok := <-msgs:
				if !ok {
					return nil
				}
				t, _ := m["type"].(string)
				logf("RECV: %s", t)
				if t == msgType {
					return m
				}
			case <-deadline:
				logf("TIMEOUT waiting for %s", msgType)
				return nil
			}
		}
	}

	drain := func(d time.Duration) {
		deadline := time.After(d)
		for {
			select {
			case m, ok := <-msgs:
				if !ok {
					return
				}
				t, _ := m["type"].(string)
				logf("RECV(drain): %s", t)
			case <-deadline:
				return
			}
		}
	}

	// 1. Join
	send(map[string]string{"type": "join", "nickname": *nickname})
	if m := waitFor("joined", 5*time.Second); m == nil {
		log.Fatal("failed to join")
	}
	logf("Joined successfully")
	drain(1 * time.Second)

	// 2. Host waits for players and starts game
	if *isHost {
		waitSec := *numPlayers*3 + 2
		logf("Waiting %d seconds for players to join...", waitSec)
		drain(time.Duration(waitSec) * time.Second)
		send(map[string]string{"type": "start_game", "themeKeyword": "mystery mansion"})
		logf("Sent start_game")
	}

	// 3. Wait for briefings
	if waitFor("briefing_public", 30*time.Second) == nil {
		log.Fatal("no briefing_public")
	}
	logf("Got briefing_public")

	if waitFor("briefing_private", 10*time.Second) == nil {
		log.Fatal("no briefing_private")
	}
	logf("Got briefing_private")

	// 4. Ready
	time.Sleep(500 * time.Millisecond)
	send(map[string]string{"type": "ready"})
	logf("Sent ready")

	if waitFor("game_started", 15*time.Second) == nil {
		log.Fatal("no game_started")
	}
	logf("Game started!")

	// 5. Chat
	time.Sleep(500 * time.Millisecond)
	send(map[string]interface{}{"type": "chat", "content": fmt.Sprintf("Hello! I'm %s.", *nickname)})
	drain(1 * time.Second)

	// 6. Examine
	send(map[string]string{"type": "examine"})
	waitFor("game_event", 10*time.Second)
	logf("Got examine result")

	// 7. Move
	time.Sleep(500 * time.Millisecond)
	send(map[string]string{"type": "move", "targetRoomId": "room-library"})
	drain(2 * time.Second)
	logf("Moved to library")

	// 8. Move back and talk to NPC
	send(map[string]string{"type": "move", "targetRoomId": "room-foyer"})
	drain(2 * time.Second)
	send(map[string]interface{}{"type": "talk", "npcId": "npc-butler", "message": "What do you know?"})
	drain(3 * time.Second)
	logf("Talked to NPC")

	// 9. End game
	if *isHost {
		time.Sleep(1 * time.Second)
		send(map[string]string{"type": "propose_end"})
		logf("Proposed end")
	} else {
		// Wait for end proposal
		time.Sleep(2 * time.Second)
		send(map[string]interface{}{"type": "end_vote", "agree": true})
		logf("Voted to end")
	}

	// 10. Wait for finish
	deadline := time.After(30 * time.Second)
	finished := false
	for !finished {
		select {
		case m, ok := <-msgs:
			if !ok {
				logf("Connection closed")
				finished = true
				break
			}
			t, _ := m["type"].(string)
			logf("RECV(end): %s", t)
			if t == "game_finished" {
				finished = true
				logf("GAME FINISHED!")
			}
		case <-deadline:
			logf("TIMEOUT waiting for game_finished")
			finished = true
		}
	}

	// Output result
	fmt.Println("OK")
}
