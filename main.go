/*
V1: Make sure 2 peers can send and receive files - Use TCP --DONE
V2: Reciprocating, Choking and Interested algorithm
V3: Peer discovery & Kademlia DHT
V4: NAT Traversals
V5: File encryption
*/
package main

import (
	"log"
	"p2p/config"
	"p2p/network"
)

func main() {
	networkConfig := config.DefaultNetworkConfig()
	peer := network.NewPeer(networkConfig)

	// Option 1: Listen for incoming connections
	// go func() {
	// 	if err := peer.Listen(); err != nil {
	// 		log.Fatalf("Listening error: %v", err)
	// 	}
	// }()

	// Option 2: Send a file to a specific peer
	// go func() {
	// 	err := peer.DialAndSendFile("192.168.1.166:5555", "C:/Users/linkp/OneDrive/Desktop/decentShare/main.txt")
	// 	if err != nil {
	// 		log.Printf("File send error: %v", err)
	// 	}
	// }()

	// Option 3: Receive a file
	go func() {
		err := peer.ReceiveFile("C:/Users/linkp/OneDrive/Desktop/decentShare/main.txt")
		if err != nil {
			log.Printf("File receive error: %v", err)
		}
	}()

	// Keep the main goroutine alive
	select {}
}
