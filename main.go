/*
V1: Make sure 2 peers can send and receive files - Use TCP --DONE
V2: Reciprocating, Choking and Interested algorithm --DONE
V3: Peer discovery & Kademlia DHT
V4: NAT Traversals
V5: File encryption
*/
package main

import (
	"context"
	"log"
	"p2p/file"
	"p2p/network"
	"time"
)

func main() {

	ExampleUsage()
	// // Listen for incoming connections
	// go func() {
	// 	if err := peer.ListenTCP(); err != nil {
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
	// wg.Add(1)
	// go func() {
	// 	err := peer.ReceiveFile("C:/Users/linkp/OneDrive/Desktop/decentShare/main.txt")
	// 	if err != nil {
	// 		log.Printf("File receive error: %v", err)
	// 	}
	// 	wg.Done()
	// }()
	// peer.CloseAllConnections()
	// wg.Wait()
}

func ExampleUsage() {
	path := "D:/Disk D files/[Nep_Blanc] Death Note [1080p] [x265] [10Bit] [Dual Audio] [Subbed] [Small]/[Nep_Blanc] Death Note 01 .mkv"
	newFile, err := file.NewFile(path)
	if err != nil {
		log.Panicln("error creating a new file: ", err)
	}
	ctx := context.Background()
	cfg := network.PeerConfig{
		Host:              "localhost",
		Port:              8080,
		MaxConnections:    50,
		ConnectionTimeout: 30 * time.Second,
		BlockSize:         16384,
	}

	// Create a new peer
	peer := network.NewPeer(cfg, newFile, network.TypeLeecher)

	// Start the peer (including connection manager)
	if err := peer.Start(ctx); err != nil {
		log.Fatalf("Failed to start peer: %v", err)
	}

	// Connect to a remote peer
	_, err = peer.ConnectToPeer(ctx, "192.168.1.166.:8080")
	if err != nil {
		log.Printf("Failed to connect to peer: %v", err)
		return
	}

	// Get connection stats
	stats := peer.GetPeerStats()
	log.Printf("Peer stats: %+v", stats)
}
