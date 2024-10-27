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
	ID          string
	Content     string
	SubChannels map[string]string
}

var tunnels = make(map[string]*Tunnel)
var tunnelsMutex = &sync.Mutex{}
var clients = make(map[string]map[string][]chan string)
var clientsMutex = &sync.Mutex{}

func main() {
	log.Println("Starting server on port 2427")
	http.HandleFunc("/", withCORS(homePage))
	http.HandleFunc("/api/v2/tunnel/create", withCORS(createTunnel))
	http.HandleFunc("/api/v2/tunnel/stream", withCORS(streamTunnelContent))
	http.HandleFunc("/api/v2/tunnel/send", withCORS(sendToTunnel))
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
	var requestData struct {
		ID         string `json:"id"`
		Content    string `json:"content"`
		SubChannel string `json:"subChannel"`
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

	if requestData.SubChannel == "" {
		http.Error(w, "No subChannel has been provided.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[requestData.ID]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnel.SubChannels[requestData.SubChannel] = requestData.Content
	tunnelsMutex.Unlock()

	clientsMutex.Lock()
	for _, client := range clients[requestData.ID][requestData.SubChannel] {
		client <- requestData.Content
	}
	clientsMutex.Unlock()

	log.Printf("Tunnel %s subChannel %s has been updated.", requestData.ID, requestData.SubChannel)
	fmt.Fprintf(w, "Tunnel %s subChannel %s has been updated.", requestData.ID, requestData.SubChannel)
}

func streamTunnelContent(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	subChannel := r.URL.Query().Get("subChannel")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		return
	}
	if subChannel == "" {
		http.Error(w, "No subChannel has been provided.\nPlease use ?subChannel= to include the subChannel.", http.StatusBadRequest)
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
	if clients[tunnelId] == nil {
		clients[tunnelId] = make(map[string][]chan string)
	}
	clients[tunnelId][subChannel] = append(clients[tunnelId][subChannel], clientChan)
	clientsMutex.Unlock()

	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			clientsMutex.Lock()
			for i, client := range clients[tunnelId][subChannel] {
				if client == clientChan {
					clients[tunnelId][subChannel] = append(clients[tunnelId][subChannel][:i], clients[tunnelId][subChannel][i+1:]...)
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
	tunnels[tunnelId] = &Tunnel{ID: tunnelId, Content: "", SubChannels: make(map[string]string)}
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
