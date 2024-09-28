package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

var publicDataBase *badger.DB
var serverDataBase *badger.DB
var timeToLive = 5       // in minutes
var cleanUpInterval = 10 // in minutes

func main() {
	http.HandleFunc("/", withCORS(homePage))
	http.HandleFunc("/tunnel", withCORS(getTunnel))
	http.HandleFunc("/tunnel/create", withCORS(createTunnel))
	http.HandleFunc("/tunnel/delete", withCORS(deleteTunnel))
	http.HandleFunc("/tunnel/send", withCORS(sendToTunnel))

	var err error
	publicDataBase, err = badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}
	serverDataBase, err = badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}
	defer publicDataBase.Close()
	defer serverDataBase.Close()

	go startCleanupTicker()

	log.Fatal(http.ListenAndServe(":8080", nil))
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

func startCleanupTicker() {
	ticker := time.NewTicker(time.Duration(cleanUpInterval) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanUp()
	}
}

func cleanUp() {
	// clean up expired tunnels
	publicDataBase.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			if item.ExpiresAt() <= uint64(time.Now().Unix()) {
				err := txn.Delete(item.Key())
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func hashToken(token string) string {
	sha512Hasher := sha512.New()
	sha512Hasher.Write([]byte(token))
	return hex.EncodeToString(sha512Hasher.Sum(nil))
}

func sendToTunnel(w http.ResponseWriter, r *http.Request) {
	// Send data to a tunnel
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

	err := publicDataBase.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(tunnelId))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			// update the value of the tunnel with the data provided
			entry := badger.NewEntry([]byte(tunnelId), []byte(content)).WithTTL(time.Duration(timeToLive) * time.Minute)
			err2 := txn.SetEntry(entry)
			if err2 != nil {
				return err2
			}
			return nil
		})
	})

	if err != nil {
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
	} else {
		log.Printf("Tunnel %s has been updated.", tunnelId)
		fmt.Fprintf(w, "Tunnel %s has been updated.", tunnelId)
	}

}

func getTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		return
	}

	err := publicDataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(tunnelId))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			// return the value of the tunnel in json format
			w.Header().Set("Content-Type", "application/json")
			response, err := json.Marshal(map[string]string{"id": tunnelId, "content": string(val)})
			if err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return err
			}
			w.Write(response)
			log.Printf("Tunnel %s has been accessed.", tunnelId)
			return nil
		})
	})
	if err != nil {
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
	}
}

func deleteTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	authToken := r.URL.Query().Get("auth")
	hashedAuthToken := hashToken(authToken)

	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		log.Println("No tunnel id provided")
		return
	}

	if authToken == "" {
		http.Error(w, "No authentication token has been provided.\nPlease use ?auth= to include the authentication token.", http.StatusBadRequest)
		log.Println("No authentication token provided")
		return
	}

	// Check if the authentication token is correct
	err := serverDataBase.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(tunnelId))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			if string(val) != hashedAuthToken {
				log.Printf("Incorrect authentication token for tunnel %s.", tunnelId)
				return fmt.Errorf("incorrect authentication token")
			}
			return nil
		})
	})

	if err != nil {
		if err.Error() == "incorrect authentication token" {
			http.Error(w, "Incorrect authentication token.", http.StatusUnauthorized)
			return
		}
		http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
		log.Printf("No tunnel with id %s exists: %v", tunnelId, err)
		return
	}

	err2 := publicDataBase.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(tunnelId))
	})

	if err2 != nil {
		http.Error(w, "Unable to delete tunnel from public database.", http.StatusInternalServerError)
		log.Printf("Unable to delete tunnel %s from public database: %v", tunnelId, err2)
		return
	}

	err3 := serverDataBase.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(tunnelId))
	})

	if err3 != nil {
		http.Error(w, "Unable to delete tunnel from server database.", http.StatusInternalServerError)
		log.Printf("Unable to delete tunnel %s from server database: %v", tunnelId, err3)
		return
	}

	log.Printf("Tunnel %s has been deleted.", tunnelId)
	fmt.Fprintf(w, "Tunnel %s has been deleted.", tunnelId)
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	tunnelId := rand.Int()
	authToken := rand.Int()
	hashedAuthToken := hashToken(fmt.Sprintf("%d", authToken))

	err := publicDataBase.Update(func(txn *badger.Txn) error {
		ttl := time.Duration(timeToLive) * time.Minute // Define the TTL duration
		entry := badger.NewEntry([]byte(fmt.Sprintf("%d", tunnelId)), []byte("")).WithTTL(ttl)
		return txn.SetEntry(entry)
	})

	if err != nil {
		log.Printf("Error updating public database: %v", err)
		http.Error(w, "Unable to create tunnel in public database.", http.StatusInternalServerError)
		return
	}

	err2 := serverDataBase.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fmt.Sprintf("%d", tunnelId)), []byte(hashedAuthToken))
	})

	if err2 != nil {
		log.Printf("Error updating server database: %v", err2)
		http.Error(w, "Unable to create tunnel in server database.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(map[string]string{"id": fmt.Sprintf("%d", tunnelId), "auth": fmt.Sprintf("%d", authToken)})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(response)
	log.Printf("Tunnel %d has been created.", tunnelId)
}
