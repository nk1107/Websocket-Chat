package main

import (
	"fmt"
	"log"
	"net/http"
)

func serveIndex(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("%s", r.URL)
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "templates/index.html")
}

func main() {

	hub := &Hub{
		clients:    map[*Client]bool{},
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go hub.Run()
	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	fmt.Printf("Listening on port 3000\n")
	log.Fatal(http.ListenAndServe(":3000", nil))

}
