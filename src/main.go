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
	Type     string   `json:"type"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Room     string   `json:"room"`
	Users    []string `json:"users"`
	Rooms    []string `json:"rooms"`
	Message  string   `json:"message"`
}

type ChatRoom struct {
	Name       string
	Clients    map[string]Client
	ClientsMtx sync.Mutex
	Queue      chan Message
}

func (cr *ChatRoom) Init() {
	cr.Queue = make(chan Message, 5)
	cr.Clients = make(map[string]Client)

	go func() {
		for {
			cr.Broadcast()
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (cr *ChatRoom) Join(client Client) {
	defer cr.ClientsMtx.Unlock()

	cr.ClientsMtx.Lock()
	if _, exists := cr.Clients[client.Username]; exists {
		log.Println("Client", client.Username, "already in chatroom")
	} else {
		cr.Clients[client.Username] = client
		msg := Message{Type: "message", Email: "Chatbot", Username: "Chatbot", Message: "<B>" + client.Username + "</B> has joined the chat."}

		cr.AddMsg(msg)
	}

}

func (cr *ChatRoom) Leave(username string) {
	cr.ClientsMtx.Lock()
	delete(cr.Clients, username)
	cr.ClientsMtx.Unlock()
	msg := Message{Type: "message", Email: "Chatbot", Username: "Chatbot", Message: "<B>" + username + "</B> has left the chat."}
	cr.AddMsg(msg)
}

func (cr *ChatRoom) AddMsg(msg Message) {
	keys := make([]string, 0, len(chatrooms))
	for k := range chatrooms {
		keys = append(keys, k)
	}
	msg.Rooms = keys
	msg.Users = cr.GetClients()
	cr.Queue <- msg
}

func (cr *ChatRoom) Broadcast() {
	m := <-cr.Queue
	for _, client := range cr.Clients {
		client.Send(m)
	}
}

func (cr *ChatRoom) GetClients() []string {
	keys := make([]string, 0, len(cr.Clients))
	for k := range cr.Clients {
		keys = append(keys, k)
	}
	return keys
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

var chatrooms = make(map[string]ChatRoom)

func main() {
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", handleConnections)

	chat := ChatRoom{Name: "main"}
	chat.Init()
	chatrooms["main"] = chat

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getInitMessage() Message {
	msg := Message{Type: "message", Email: "Chatbot", Username: "Chatbot"}
	return msg
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	var chat ChatRoom
	if _, exists := chatrooms["main"]; !exists {
		chat := ChatRoom{Name: "main"}
		chat.Init()
		chatrooms["main"] = chat
	}
	chat = chatrooms["main"]

	client := Client{Conn: ws, BelongsTo: &chat}
	for {
		var msg Message

		err := ws.ReadJSON(&msg)
		log.Println(msg)
		if msg.Type == "join" {
			if client.BelongsTo.Name != msg.Room {
				client.Exit()
			}
			chat := chatrooms[msg.Room]
			client.BelongsTo = &chat
			chat.Join(client)
		} else if msg.Type == "message" {
			client.NewMsg(msg)
		} else if msg.Type == "createUser" {
			client.Init(msg.Username, msg.Email)
		} else if msg.Type == "createRoom" {
			chat := ChatRoom{Name: msg.Room}
			chat.Init()
			chatrooms[msg.Room] = chat
		}
		if err != nil {
			log.Printf("error: %v", err)
			client.Exit()
			break
		}
	}
}
