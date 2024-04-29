package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

// Server Define a struct to represent the WebSocket server.
type Server struct {
	clients    map[string]*websocket.Conn
	broadcast  chan Message
	register   chan RegisterMessage
	unregister chan string
}

// Message Define a struct to represent a message sent over WebSocket.
type Message struct {
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Content    any    `json:"content"`
}

// RegisterMessage Define a struct for registering clients.
type RegisterMessage struct {
	ClientID string `json:"client_id"`
	Conn     *websocket.Conn
}

func main() {
	// Initialize the server.
	server := &Server{
		clients:    make(map[string]*websocket.Conn),
		broadcast:  make(chan Message),
		register:   make(chan RegisterMessage),
		unregister: make(chan string),
	}

	// Handle WebSocket connections.
	go server.handleMessages()

	// Set up HTTP server to handle WebSocket connections.
	http.HandleFunc("/ws", server.handleWebSocket)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (server *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all connections
			return true
		},
	}

	// Upgrade HTTP connection to WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()

	// Register client.
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		log.Println("Client ID not provided.")
		return
	}

	server.register <- RegisterMessage{ClientID: clientID, Conn: conn}

	// Listen for messages from the client.
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			return
		}

		// Handle received message
		// You may implement logic here to handle different types of messages
		if clientID != msg.ReceiverID {
			server.broadcast <- Message{SenderID: clientID, ReceiverID: msg.ReceiverID, Content: msg.Content}
		}
	}
}

func (server *Server) handleMessages() {
	for {
		select {
		case message := <-server.broadcast:
			// Send message to the specified client.
			receiverConn, ok := server.clients[message.ReceiverID]
			if !ok {
				log.Printf("Client with ID %s not found\n", message.ReceiverID)
				continue
			}
			err := receiverConn.WriteJSON(message)
			if err != nil {
				log.Println(err)
				continue
			}
		case registration := <-server.register:
			// Register client.
			server.clients[registration.ClientID] = registration.Conn
			log.Printf("Client with ID %s registered\n", registration.ClientID)
		case clientID := <-server.unregister:
			// Unregister client.
			delete(server.clients, clientID)
			log.Printf("Client with ID %s unregistered\n", clientID)
		}
	}
}
