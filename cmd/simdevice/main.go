package main

import (
	"flag"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "ws://127.0.0.1:8080/ws", "server WebSocket URL")
	token := flag.String("token", "local-test-token-123456", "pairing token")
	flag.Parse()
	c, _, err := websocket.DefaultDialer.Dial(*url, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if err := c.WriteJSON(map[string]any{"type": "pair", "token": *token, "deviceId": "simulated-phone", "model": "KMoni Simulator", "android": "35", "app": "simulator", "rooted": true}); err != nil {
		log.Fatal(err)
	}
	var response map[string]string
	if err := c.ReadJSON(&response); err != nil {
		log.Fatal(err)
	}
	log.Printf("server response: %v", response)
	for {
		if err := c.WriteJSON(map[string]any{"type": "status", "rooted": true}); err != nil {
			log.Fatal(err)
		}
		log.Printf("status sent")
		time.Sleep(10 * time.Second)
	}
}
