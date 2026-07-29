package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type IncomingMessage struct {
	Type string `json:"type"`
	// OneOf: Type decides which field is used
	Name string `json:"name,omitempty"` // register
	Text string `json:"text,omitempty"` // message
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)
var mutex = &sync.Mutex{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}

	defer conn.Close()

	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}

		var receivedMsgJson = new(IncomingMessage)
		parsingErr := json.Unmarshal(message, &receivedMsgJson)
		if parsingErr != nil {
			fmt.Println("Error parsing received message to json:", parsingErr)
		}

		switch receivedMsgJson.Type {
		case "register":
			{
				// TODO: add to the list of clients
			}
		case "message":
			{
				// TODO: add to broadcast channel
			}
		default:
			{
				fmt.Println("unrecognized message type:", receivedMsgJson.Type)
			}
		}

		broadcast <- message
	}
}

func broadcastMessages() {
	for {
		message := <-broadcast

		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	go broadcastMessages()
	fmt.Println("WebSocket server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
