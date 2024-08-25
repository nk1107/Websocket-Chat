package main

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	pongWait       = 60 * time.Second
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	id       uuid.UUID
	username string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
}

func (c *Client) ReadPump() {
	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, text, err := c.conn.ReadMessage()
		log.Printf("Value %v", string(text))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}

		msg := &WsMessage{}
		reader := bytes.NewReader(text)
		decoder := json.NewDecoder(reader)
		err = decoder.Decode(msg)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		log.Printf("converted text %v", *msg)
		c.hub.broadcast <- &Message{ClientId: c.id, ClientUsername: c.username, Text: msg.Text}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			log.Printf("Recieved message in the send channel: %v", string(message))
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	id := uuid.New()
	username := "user" + strconv.Itoa(rand.Intn(100)+1)
	client := &Client{
		id:       id,
		username: username,
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte),
	}

	client.hub.register <- client
	go client.ReadPump()
	go client.WritePump()

}
