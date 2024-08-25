package main

import (
	"bytes"
	"github.com/google/uuid"
	"html/template"
	"log"
)

type Message struct {
	ClientUsername string
	ClientId       uuid.UUID
	Text           string
}
type WsMessage struct {
	Headers interface{} `json:"HEADERS"`
	Text    string      `json:"message"`
}

type Hub struct {
	clients    map[*Client]bool
	Messages   []*Message
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client has been registered: %v", client.id)
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
				log.Printf("Client has been unregistered: %v", client.id)
			}
		case msg := <-h.broadcast:
			h.Messages = append(h.Messages, msg)
			for client := range h.clients {
				select {
				case client.send <- getMessageTemplate(msg):
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func getMessageTemplate(msg *Message) []byte {
	templ, err := template.ParseFiles("templates/message.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
	}
	var rendered bytes.Buffer
	err = templ.Execute(&rendered, msg)
	if err != nil {
		log.Printf("Error rendering template: %v", err)
	}
	return rendered.Bytes()
}
