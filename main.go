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
}

func ExampleUsage() {
	path := "C:\\Users\\linkp\\Downloads\\archlinux-2024.11.01-x86_64.iso"
	newFile, err := file.NewFile(path)
	if err != nil {
		log.Panicln("error creating a new file: ", err)
	}
	ctx := context.Background()
	cfg := network.PeerConfig{
		Host:              "192.168.1.166",
		Port:              8080,
		MaxConnections:    50,
		ConnectionTimeout: 30 * time.Second,
		BlockSize:         16384,
	}

	peer := network.NewPeer(cfg, newFile, network.TypeSeeder)

	if err := peer.Start(ctx); err != nil {
		log.Fatalf("Failed to start peer: %v", err)
	}

	_, err = peer.ConnectToPeer(ctx, "192.168.1.166.:8080")
	if err != nil {
		log.Printf("Failed to connect to peer: %v", err)
		return
	}

	stats := peer.GetPeerStats()
	log.Printf("Peer stats: %+v", stats)
}
