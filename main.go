package main

import (
	"fmt"
	"p2p/cmd"
	"p2p/config"
	"p2p/connection"
	"p2p/db"
	"p2p/model"
)

var GlobalHashTable model.Tracker

//7mwHbbwTD0WZucsI
//linkpranayv90

func main() {
	config.LoadConfig()
	db.InitDB()
	GlobalHashTable.Table = map[string]map[string]string{}
	ip := cmd.GetIP()
	fmt.Println(ip)
	// now check what file is the user looking for
	// let's suppose user is looking to download a file named "gopher.png"
	// reading the file will be a blocking call
	// this filename will be hashed using a sha1 hash function - 'c1e48612f7cd55aa606c3b23ae60c44656f9443d'
	// maintain a global hash table (tracker)
	sha1 := "c1e48612f7cd55aa606c3b23ae60c44656f9443d"
	peerPool, exists := GlobalHashTable.Table[sha1]
	if !exists {
		// add the sha1 hash to the map
		fmt.Println("adding the hash to the map")
		GlobalHashTable.Table[sha1] = map[string]string{ip.String(): "peer1"}
		fmt.Println(GlobalHashTable.Table)
	}
	for peer := range peerPool {
		fmt.Println(peer)
	}

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
	connection.StartTCPconnection()
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
