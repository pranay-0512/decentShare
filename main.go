/*
V1: Make sure 2 peers can send and receive files - Use TCP --DONE
V2: Reciprocating, Choking and Interested algorithm --DONE, what is left -- piece selection algorithm -> random first policy, rarest first policy, strict priority policy --DONE
V3: Peer discovery & Kademlia DHT --DONE
V4: NAT Traversals
V5: File encryption
*/

/*
	issues i am facing
	1. NAT traversals- STUN servers use UDP hole punching, and the port opening does not allow for incoming TCP packets. I have to either limit this to local network or use UDP instead (with extra steps)

	what is remaining:
	1. File and Piece SHA1 hash verify --DONE

*/

package main

import (
	"fmt"
	"log"
	"p2p/config"
	"p2p/file"
	"path/filepath"
	"sync"

	"github.com/sqweek/dialog"
)

func main() {
	ExampleUsage()
}

func ExampleUsage() {
	var wg sync.WaitGroup
	path, err := dialog.File().Title("Select a file").Load()
	if err != nil {
		log.Panicln("error selecting a destination folder: ", err)
		return
	}
	destPath, err := dialog.Directory().Title("Select a destination folder").Browse()
	absPath := filepath.Join(destPath, "decent/completed")
	config.SetDestPath(absPath)
	fmt.Println("Chosen Destination path: ", absPath)
	if err != nil {
		log.Panicln("error selecting a file: ", err)
		return
	}
	newFile, err := file.NewFile(path)
	if err != nil {
		log.Panicln("error creating a new file: ", err)
	}

	errChan := make(chan error, newFile.Pieces)
	wg.Add(1)
	go func() {
		defer wg.Done()
		newFile.Chunkify(errChan)
		newFile.Merge(file.TempDst, errChan)
	}()
	wg.Wait()
	close(errChan)
	fmt.Println("File saved at: ", config.GetDestPath())

	ctx := context.Background()
	cfg := network.PeerConfig{
		Host:              "192.168.29.119",
		Port:              8080,
		MaxConnections:    50,
		ConnectionTimeout: 30 * time.Second,
		BlockSize:         16384,
	}

	peer := network.New(cfg, newFile, network.TypeSeeder)

	if err := peer.Start(ctx); err != nil {
		log.Fatalf("Failed to start peer: %v", err)
	}

	_, err = peer.ConnectToPeer(ctx, "192.168.29.119.:8080")
	if err != nil {
		log.Printf("Failed to connect to peer: %v", err)
		return
	}

	stats := peer.GetPeerStats()
	log.Printf("Peer stats: %+v", stats)

}
