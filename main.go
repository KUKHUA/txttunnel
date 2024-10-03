package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
)

type Tunnel struct {
	ID      string
	Content string
}

var tunnels = make(map[string]*Tunnel)
var tunnelsMutex = &sync.Mutex{}
var clients = make(map[string][]chan string)
var clientsMutex = &sync.Mutex{}

func main() {
	http.HandleFunc("/", withCORS(homePage))
	http.HandleFunc("/tunnel/stream", withCORS(getTunnel))
	http.HandleFunc("/tunnel/create", withCORS(createTunnel))
	http.HandleFunc("/tunnel/send", withCORS(sendToTunnel))
	http.HandleFunc("/tunnel/send/post", withCORS(sendToTunnelPost))

	log.Fatal(http.ListenAndServe(":2427", nil))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the TXTTunnel homepage!")
}

func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func sendToTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	content := r.URL.Query().Get("content")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		return
	}
	if content == "" {
		http.Error(w, "No content has been provided.\nPlease use ?content= to include the content.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[tunnelId]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnel.Content = content
	tunnelsMutex.Unlock()

	clientsMutex.Lock()
	for _, client := range clients[tunnelId] {
		client <- content
	}
	clientsMutex.Unlock()

	log.Printf("Tunnel %s has been updated.", tunnelId)
	fmt.Fprintf(w, "Tunnel %s has been updated.", tunnelId)
}

func sendToTunnelPost(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	if requestData.ID == "" {
		http.Error(w, "No tunnel id has been provided.", http.StatusBadRequest)
		return
	}
	if requestData.Content == "" {
		http.Error(w, "No content has been provided.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[requestData.ID]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnel.Content = requestData.Content
	tunnelsMutex.Unlock()

	clientsMutex.Lock()
	for _, client := range clients[requestData.ID] {
		client <- requestData.Content
	}
	clientsMutex.Unlock()

	log.Printf("Tunnel %s has been updated.", requestData.ID)
	fmt.Fprintf(w, "Tunnel %s has been updated.", requestData.ID)
}

func getTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	_, exists := tunnels[tunnelId]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnelsMutex.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan string)
	clientsMutex.Lock()
	clients[tunnelId] = append(clients[tunnelId], clientChan)
	clientsMutex.Unlock()

	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			clientsMutex.Lock()
			for i, client := range clients[tunnelId] {
				if client == clientChan {
					clients[tunnelId] = append(clients[tunnelId][:i], clients[tunnelId][i+1:]...)
					break
				}
			}
			clientsMutex.Unlock()
			return
		}
	}
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := fmt.Sprintf("%02d%c%02d%c%02d%c", rand.Intn(100), 'A'+rune(rand.Intn(26)), rand.Intn(100), 'A'+rune(rand.Intn(26)), rand.Intn(100), 'A'+rune(rand.Intn(26)))
	tunnelsMutex.Lock()
	tunnels[tunnelId] = &Tunnel{ID: tunnelId, Content: ""}
	tunnelsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(map[string]string{"id": tunnelId})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(response)
	log.Printf("Tunnel %s has been created.", tunnelId)
}
