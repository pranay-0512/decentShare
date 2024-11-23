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
	// networkConfig := config.DefaultNetworkConfig()
	// peer := network.NewPeer(networkConfig)
	// wg := sync.WaitGroup{}
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

	path := "D:/Disk D files/[Nep_Blanc] Death Note [1080p] [x265] [10Bit] [Dual Audio] [Subbed] [Small]/[Nep_Blanc] Death Note 01 .mkv"
	newFile, err := file.NewFile(path)
	if err != nil {
		log.Panicln("error creating a new file: ", err)
	}
	chunks, err := newFile.Chunkify()
	if err != nil {
		log.Panicln("error chunkifying file: ", err)
	}
	err = newFile.Merge(chunks)
	if err != nil {
		log.Panicln("error merging file: ", err)
	}
}
