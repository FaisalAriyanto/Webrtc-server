package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"example.com/m/v2/turn"

	socketio "github.com/googollee/go-socket.io"
	redis "github.com/redis/go-redis/v9"
)

type SignalMessage struct {
	Type      string                 `json:"type"`
	SDP       string                 `json:"sdp"`
	Target    string                 `json:"target"`
	From      string                 `json:"from"`
	Candidate map[string]interface{} `json:"candidate"`
}

var userConnections = make(map[string]socketio.Conn)
var ctx = context.Background()
var redisClient *redis.Client

func main() {
	log.SetOutput(os.Stdout)
	log.Println("Program started")

	server := socketio.NewServer(nil)
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Printf("Connected: %s", s.ID())
		s.Join("all")
		log.Printf("Before broadcasting user_connected: %s", s.ID())
		go func() {
			userConnections[s.ID()] = s
			go broadcastUserList(server)
			log.Printf("After broadcasting user_connected: %s", s.ID())
		}()

		return nil
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Printf("Disconnected: %s", s.ID())
		log.Printf("Before broadcasting user_disconnected: %s", s.ID())
		go func() {
			delete(userConnections, s.ID()) // Remove the user from userConnections
			broadcastUserList(server)       // Call broadcastUserList after removing the user
			log.Printf("After broadcasting user_disconnected: %s", s.ID())
		}()
		s.Leave("all")
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("Error:", e)
	})

	server.OnEvent("/", "offer", func(s socketio.Conn, msg string) {
		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling offer message: %v", err)
			return
		}
		log.Printf("Received offer from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("offer", msg)
		}
	})

	server.OnEvent("/", "answer", func(s socketio.Conn, msg string) {
		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling answer message: %v", err)
			return
		}
		log.Printf("Received answer from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("answer", msg)
		}
	})

	server.OnEvent("/", "ice_candidate", func(s socketio.Conn, msg string) {
		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling ice_candidate message: %v", err)
			return
		}
		log.Printf("Received ice_candidate from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("ice_candidate", msg)
		}
	})

	// Add the initiate_call event handler
	server.OnEvent("/", "initiate_call", func(s socketio.Conn, msg string) {
		log.Printf("Received initiate_call ")

		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling initiate_call message: %v", err)
			return
		}
		log.Printf("Received initiate_call from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("initiate_call", msg)
		}
	})

	// Add the initiate_call event handler
	server.OnEvent("/", "start_call", func(s socketio.Conn, msg string) {
		log.Printf("Received calling ")

		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling calling message: %v", err)
			return
		}
		message.From = s.ID()
		log.Printf("Received calling from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("called", message)
		}
	})

	// Add the initiate_call event handler
	server.OnEvent("/", "accept", func(s socketio.Conn, msg string) {
		log.Printf("Received accept ")

		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling accept message: %v", err)
			return
		}
		log.Printf("Received accept from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("accept", msg)
		}
	})

	// Add the end_call event handler
	server.OnEvent("/", "end_call", func(s socketio.Conn, msg string) {
		log.Printf("Received end_call")

		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling end_call message: %v", err)
			return
		}
		log.Printf("Received end_call from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("end_call", msg)
		}
	})

	// Add the end_call event handler
	server.OnEvent("/", "remote_effect", func(s socketio.Conn, msg string) {
		log.Printf("Received remote_effect")

		var message SignalMessage
		if err := json.Unmarshal([]byte(msg), &message); err != nil {
			log.Printf("Error unmarshaling end_call message: %v", err)
			return
		}
		log.Printf("Received end_call from %s to %s", s.ID(), message.Target)

		if target, ok := userConnections[message.Target]; ok {
			target.Emit("remote_effect", msg)
		}
	})

	// Create and start the TURN server
	turnConfig := turn.DefaultConfig()
	turnServer := turn.NewTurnServer(turnConfig)

	go server.Serve()
	defer server.Close()
	defer turnServer.Close()

	http.Handle("/socket.io/", server)
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func broadcastUserList(server *socketio.Server) {
	users := make([]string, 0, len(userConnections))
	for userID := range userConnections {
		users = append(users, userID)
	}
	go server.BroadcastToRoom("/", "all", "update_users", users)
}
