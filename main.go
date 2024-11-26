/*
V1: Make sure 2 peers can send and receive files - Use TCP --DONE
V2: Reciprocating, Choking and Interested algorithm --DONE
V3: Peer discovery & Kademlia DHT
V4: NAT Traversals
V5: File encryption
*/
package main

import (
	"log"
	"p2p/file"
)

func main() {
	ExampleUsage()
}

func ExampleUsage() {
	path := "D:\\Disk D files\\[Nep_Blanc] Death Note [1080p] [x265] [10Bit] [Dual Audio] [Subbed] [Small]\\[Nep_Blanc] Death Note 04 .mkv"
	newFile, err := file.NewFile(path)
	if err != nil {
		log.Panicln("error creating a new file: ", err)
	}
	err = newFile.Chunkify()
	if err != nil {
		log.Panicln("error chunkifying file: ", err)
	}
	err = newFile.Merge(file.TempDst)

	// ctx := context.Background()
	// cfg := network.PeerConfig{
	// 	Host:              "192.168.1.166",
	// 	Port:              8080,
	// 	MaxConnections:    50,
	// 	ConnectionTimeout: 30 * time.Second,
	// 	BlockSize:         16384,
	// }

	// peer := network.NewPeer(cfg, newFile, network.TypeSeeder)

	// if err := peer.Start(ctx); err != nil {
	// 	log.Fatalf("Failed to start peer: %v", err)
	// }

	// _, err = peer.ConnectToPeer(ctx, "192.168.1.166.:8080")
	// if err != nil {
	// 	log.Printf("Failed to connect to peer: %v", err)
	// 	return
	// }

	// stats := peer.GetPeerStats()
	// log.Printf("Peer stats: %+v", stats)
}
