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
	http.HandleFunc("/LICENSE", withCORS(giveLicense))
	http.HandleFunc("/api/v3/tunnel/create", withCORS(createTunnel))
	http.HandleFunc("/api/v3/tunnel/stream", withCORS(streamTunnelContent))
	http.HandleFunc("/api/v3/tunnel/get", withCORS(getTunnelContent))
	http.HandleFunc("/api/v3/tunnel/send", withCORS(sendToTunnel))
	log.Fatal(http.ListenAndServe(":2427", nil))
}

func giveLicense(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/LICENSE.txt")
}

func homePage(w http.ResponseWriter, r *http.Request) {
	// Return the home page /web/index.html
	http.ServeFile(w, r, "web/index.html")
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

func getTunnelContent(w http.ResponseWriter, r *http.Request) {
	tunnelId := ""
	subChannel := ""
	if r.Method == http.MethodGet {
		tunnelId = r.URL.Query().Get("id")
		subChannel = r.URL.Query().Get("subChannel")
		if r.URL.Query().Get("subchannel") != "" {
			subChannel = r.URL.Query().Get("subchannel")
		}
		if r.URL.Query().Get("ID") != "" {
			tunnelId = r.URL.Query().Get("ID")
		}
	} else if r.Method == http.MethodPost {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body", http.StatusInternalServerError)
			return
		}

		// get 'id' from the request body JSON
		var requestBodyJSON map[string]string
		err = json.Unmarshal(requestBody, &requestBodyJSON)
		if err != nil {
			http.Error(w, "Failed to parse the request body", http.StatusInternalServerError)
			return
		}

		if requestBodyJSON["subchannel"] != "" {
			subChannel = requestBodyJSON["subchannel"]
		}

		if requestBodyJSON["subChannel"] != "" {
			subChannel = requestBodyJSON["subChannel"]
		}

		if requestBodyJSON["ID"] != "" {
			tunnelId = requestBodyJSON["ID"]
		}

		if requestBodyJSON["id"] != "" {
			tunnelId = requestBodyJSON["id"]
		}
	}

	if subChannel == "" {
		subChannel = "main"
	}

	if tunnelId == "" {
		http.Error(w, "The request must contain a valid 'id' parameter or field", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[tunnelId]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
		return
	}
	// Return the content of the subchannel if it exists
	if tunnel.SubChannels[subChannel] != "" {
		w.Header().Set("Content-Type", "application/json")
		response, err := json.Marshal(map[string]string{"content": tunnel.SubChannels[subChannel]})
		if err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		w.Write(response)
	}
	tunnelsMutex.Unlock()
}

func streamTunnelContent(w http.ResponseWriter, r *http.Request) {
	tunnelId := ""
	subChannel := ""
	if r.Method == http.MethodGet {
		tunnelId = r.URL.Query().Get("id")
		subChannel = r.URL.Query().Get("subChannel")
		if r.URL.Query().Get("subchannel") != "" {
			subChannel = r.URL.Query().Get("subchannel")
		}
		if r.URL.Query().Get("ID") != "" {
			tunnelId = r.URL.Query().Get("ID")
		}
	} else if r.Method == http.MethodPost {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body", http.StatusInternalServerError)
			return
		}

		// get 'id' from the request body JSON
		var requestBodyJSON map[string]string
		err = json.Unmarshal(requestBody, &requestBodyJSON)
		if err != nil {
			http.Error(w, "Failed to parse the request body", http.StatusInternalServerError)
			return
		}

		if requestBodyJSON["subchannel"] != "" {
			subChannel = requestBodyJSON["subchannel"]
		}

		if requestBodyJSON["subChannel"] != "" {
			subChannel = requestBodyJSON["subChannel"]
		}

		if requestBodyJSON["ID"] != "" {
			tunnelId = requestBodyJSON["ID"]
		}

		if requestBodyJSON["id"] != "" {
			tunnelId = requestBodyJSON["id"]
		}
	}

	if subChannel == "" {
		subChannel = "main"
	}

	if tunnelId == "" {
		http.Error(w, "The request must contain a valid 'id' parameter or field", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	_, exists := tunnels[tunnelId]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
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

func sendToTunnel(w http.ResponseWriter, r *http.Request) {
	// we need the id, subchannel and content
	if r.Method == http.MethodPost {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body", http.StatusInternalServerError)
			return
		}

		// get 'id', 'subchannel' and 'content' from the request body JSON
		var requestBodyJSON map[string]string
		err = json.Unmarshal(requestBody, &requestBodyJSON)
		if err != nil {
			http.Error(w, "Failed to parse the request body", http.StatusInternalServerError)
			return
		}

		if requestBodyJSON["subchannel"] != "" {
			requestBodyJSON["subChannel"] = requestBodyJSON["subchannel"]
		}

		if requestBodyJSON["subChannel"] == "" {
			requestBodyJSON["subChannel"] = "main"
		}

		if requestBodyJSON["ID"] != "" {
			requestBodyJSON["id"] = requestBodyJSON["ID"]
		}

		if requestBodyJSON["id"] == "" || requestBodyJSON["subChannel"] == "" || requestBodyJSON["content"] == "" {
			http.Error(w, "The request body must contain a valid 'id', 'subChannel' and 'content' field", http.StatusBadRequest)
			return
		}

		tunnelsMutex.Lock()
		tunnel, exists := tunnels[requestBodyJSON["id"]]
		if !exists {
			tunnelsMutex.Unlock()
			http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
			return
		}
		tunnel.SubChannels[requestBodyJSON["subChannel"]] = requestBodyJSON["content"]
		tunnelsMutex.Unlock()

		clientsMutex.Lock()
		for _, client := range clients[requestBodyJSON["id"]][requestBodyJSON["subChannel"]] {
			client <- requestBodyJSON["content"]
		}
		clientsMutex.Unlock()

		w.WriteHeader(http.StatusOK)
	} else if r.Method == http.MethodGet {
		id := r.URL.Query().Get("id")
		subChannel := r.URL.Query().Get("subChannel")
		content := r.URL.Query().Get("content")

		if r.URL.Query().Get("subchannel") != "" {
			subChannel = r.URL.Query().Get("subchannel")
		}

		if r.URL.Query().Get("ID") != "" {
			id = r.URL.Query().Get("ID")
		}

		if id == "" || subChannel == "" || content == "" {
			http.Error(w, "The request must contain a valid 'id', 'subChannel' and 'content' parmerters", http.StatusBadRequest)
			return
		}

		tunnelsMutex.Lock()
		tunnel, exists := tunnels[id]
		if !exists {
			tunnelsMutex.Unlock()
			http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
			return
		}
		tunnel.SubChannels[subChannel] = content
		tunnelsMutex.Unlock()

		clientsMutex.Lock()
		for _, client := range clients[id][subChannel] {
			client <- content
		}
		clientsMutex.Unlock()
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Method not allowed. Only POST requests are allowed.", http.StatusMethodNotAllowed)
		return
	}
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	// check if it's a POST or GET request
	if r.Method == http.MethodPost {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body", http.StatusInternalServerError)
			return
		}

		// get 'id' from the request body JSON
		var requestBodyJSON map[string]string
		err = json.Unmarshal(requestBody, &requestBodyJSON)
		if err != nil {
			http.Error(w, "Failed to parse the request body", http.StatusInternalServerError)
			return
		}

		if requestBodyJSON["id"] == "" {
			http.Error(w, "The request body must contain a vaild 'id' field", http.StatusBadRequest)
		}

		tunnelsMutex.Lock()
		tunnels[requestBodyJSON["id"]] = &Tunnel{ID: requestBodyJSON["id"], Content: "", SubChannels: make(map[string]string)}
		tunnelsMutex.Unlock()

		response, err := json.Marshal(map[string]string{"id": requestBodyJSON["id"]})

		if err != nil {
			http.Error(w, "Error creating the tunnel", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	} else if r.Method == http.MethodGet {
		if r.URL.Query().Get("id") == "" {
			// The user used the GET method but didn't provide an ID
			tunnelId := generateRandomID(4)
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
		} else {
			// The user used the GET method and provided an ID
			tunnelsMutex.Lock()
			tunnels[r.URL.Query().Get("id")] = &Tunnel{ID: r.URL.Query().Get("id"), Content: "", SubChannels: make(map[string]string)}
			tunnelsMutex.Unlock()
			response, err := json.Marshal(map[string]string{"id": r.URL.Query().Get("id")})
			if err != nil {
				http.Error(w, "Error creating the tunnel", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
		}
	} else {
		http.Error(w, "Method not allowed. Only POST and GET requests are allowed.", http.StatusMethodNotAllowed)
		return
	}
}

func generateRandomID(amount int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ123456789`~!@#$%^&*()_-+=[]{}|;:',.<>/?"
	b := make([]byte, amount)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
