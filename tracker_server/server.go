package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Tracker struct {
	mu    sync.Mutex
	Table map[string]map[string]string // filename -> peerIP+port ("119.82.83.111:9090") -> status
}

var tracker = Tracker{
	Table: make(map[string]map[string]string),
}

func main() {
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/delete", deleteHandler)

	fmt.Println("Server listening on port 8080")
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Missing 'filename' query parameter", http.StatusBadRequest)
		return
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	peers, ok := tracker.Table[filename]
	if !ok {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	response, _ := json.Marshal(peers)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Filename string `json:"filename"`
		PeerIP   string `json:"peer_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if request.Filename == "" || request.PeerIP == "" {
		http.Error(w, "Filename and PeerIP are required", http.StatusBadRequest)
		return
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	if tracker.Table[request.Filename] == nil {
		tracker.Table[request.Filename] = make(map[string]string)
	}
	tracker.Table[request.Filename][request.PeerIP] = "active"
	peers, ok := tracker.Table[request.Filename]
	if !ok {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	response, _ := json.Marshal(peers)
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Filename string `json:"filename"`
		PeerIP   string `json:"peer_ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if request.Filename == "" || request.PeerIP == "" {
		http.Error(w, "Filename and PeerIP are required", http.StatusBadRequest)
		return
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	peers, ok := tracker.Table[request.Filename]
	if !ok {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if _, peerExists := peers[request.PeerIP]; !peerExists {
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	delete(peers, request.PeerIP)
	if len(peers) == 0 {
		delete(tracker.Table, request.Filename) // Cleanup file entry if no peers left
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Peer removed successfully"))
}
