package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"p2p/cmd"
	"p2p/config"
	"p2p/connection"
	"p2p/model"
)

var GlobalHashTable model.Tracker

func main() {
	config.LoadConfig()
	ip := cmd.GetIP()
	fmt.Println("Public IP: ", ip)

	fileName := "file1.txt"
	res, err := http.Post("http://localhost:8080/post", "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"filename": "%s", "peer_ip": "%s"}`, fileName, ip.String()))))
	if err != nil {
		fmt.Println("Error posting to tracker", err)
	}
	body, _ := io.ReadAll(res.Body)
	fmt.Printf("Response from POST: %s\n", string(body))

	connection.StartTCPconnection()

	// when main function is called (the entry point),
	// it means the user wants to become a peer
	// meaning it will either be a leecher or a seeder
	// user can select which file it needs to seed (seeder state)
	// user will get a list of files that he can leech (leecher state)

	// once he selects the file to either seed or leech, he will talk to DHT to find out the kv store of that file.
	// meaning he will have the public ips of all the peers already in the peer pool.
	// the choking algorithm will come into play and he will be randomly unchoked.
	// before unchoking the user, a peer will setup a TCP connection with the user.
	// Then he will send a random piece of file to the user.
	// The connection will not close untill all the pieces are trasnfered. //TODO - maybe the connection will only end if a peer abruptly leaves.
	// Connection will not be closed further.
	// A peer can handle more than 20000 TCP connections (both idle and active) at once.
	// The open connection will help in sending have messages and bitfields
	//
}

/*
Implementation Steps:

	Basic Network Layer:
		Implement TCP connections
		Handle peer discovery - Trackers for now
		Basic message protocol

	Peer Management:
		Track connected peers
		Implement choking algorithm
		Handle peer states

	Piece Management:
		Split files into pieces
		Track piece availability
		Implement piece selection

	Data Transfer:
		Request/response protocol
		Block-level transfer
		Verify data integrity

	DHT Implementation:
		Node ID generation
		K-bucket management
		Implement RPCs
		Routing table maintenance

	Tracker Integration:
		Implement tracker protocol
		Handle peer lists
		Periodic updates

	File Management:
		Torrent file parsing
		File writing/reading
		Progress tracking

	Optimization:
		Implement endgame mode
		Optimize piece selection
		Fine-tune parameters

	Additional Features:
		Resume capability
		Bandwidth management
		Stats tracking
*/
