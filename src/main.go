package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{}

// Message chat message
type Message struct {
	Type     string `json:"type"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

type ChatRoom struct {
	Clients    map[string]Client
	ClientsMtx sync.Mutex
	Queue      chan Message
}

func (cr *ChatRoom) Init() {
	cr.Queue = make(chan Message, 5)
	cr.Clients = make(map[string]Client)
	log.Println("Init chatroom")

	go func() {
		for {
			log.Println("Looping")
			cr.Broadcast()
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (cr *ChatRoom) Join(client Client) {
	defer cr.ClientsMtx.Unlock()

	cr.ClientsMtx.Lock()
	if _, exists := cr.Clients[client.Username]; exists {
		log.Println("Client", client.Username, "already exists")
	}
	cr.Clients[client.Username] = client
	msg := Message{Type: "message", Username: "Chatbot", Message: "<B>" + client.Username + "</B> has joined the chat."}

	cr.AddMsg(msg)

}

func (cr *ChatRoom) Leave(username string) {
	cr.ClientsMtx.Lock()
	delete(cr.Clients, username)
	cr.ClientsMtx.Unlock()
	msg := Message{Type: "message", Username: "Chatbot", Message: "<B>" + username + "</B> has left the chat."}
	cr.AddMsg(msg)
}

func (cr *ChatRoom) AddMsg(msg Message) {
	cr.Queue <- msg
}

func (cr *ChatRoom) Broadcast() {
	m := <-cr.Queue
	log.Println("Checking", m)
	for _, client := range cr.Clients {
		client.Send(m)
	}
}

type Client struct {
	Username  string
	Email     string
	Conn      *websocket.Conn
	BelongsTo *ChatRoom
}

func (cl *Client) Init(username string, email string) {
	cl.Username = username
	cl.Email = email
}

func (cl *Client) NewMsg(msg Message) {
	cl.BelongsTo.AddMsg(msg)
}

func (cl *Client) Exit() {
	cl.BelongsTo.Leave(cl.Username)
}

func (cl *Client) Send(msg Message) {
	err := cl.Conn.WriteJSON(msg)
	if err != nil {
		log.Printf("Error: %v", err)
		cl.Exit()
	}
}

var chat ChatRoom

func main() {
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)

	chat.Init()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	client := Client{Conn: ws, BelongsTo: &chat}

	log.Println("Connection")
	for {
		var msg Message

		err := ws.ReadJSON(&msg)
		if msg.Type == "connect" {
			log.Println(msg)
			client.Init(msg.Username, msg.Email)
			chat.Join(client)
			log.Println("Client connected", client)
		} else if msg.Type == "message" {
			client.NewMsg(msg)
		}
		if err != nil {
			log.Printf("error: %v", err)
			client.Exit()
			break
		}
	}
}
