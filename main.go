/*
V1: Make sure 2 peers can send and receive files - Use TCP
V2: Reciprocating, Choking and Interested algorithm
V3: Peer discovery & Kademlia DHT
V4: NAT Traversals
V5: File encryption
*/
package main

import (
	"p2p/connection"
)

func main() {
	connection.StartTCPconnection()
}
