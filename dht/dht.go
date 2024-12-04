package dht

// kademlia
// implement a DHT finding algorithm that uses XOR_Addresses to find the nearest
// distance between different nodes.
// it implements a trie. Each trie is a bit.
// A path (1 1 0 0 1) is id of a node or a peer.

/*
	Each machine acts as a node. Each node will have 2 things. A KV store and a routing table.
	A KV store is a map of keys (file hash) and values (peer pool).
	A routing table is a trie of nodes. Each node & key will be a representation of 10bit integer.
	For example: 1010110101.
	We make sure that a key is stored in the node that is closest to the key.
	For this we will use XOR distance metric.

	Routing table:
	Each node will have 5 buckets, each buckets being the closest node to that node.
	If the k-buckets are full, ping the least recently seen node. If it pongs, discard the new node. Else expel the least recently seen node and add the new node to the tail.

	4 main operations:
	PING: Check health of a node.const
	FIND_NODE: Find a node in the network. A node will respond with the k closest nodes to the target node.
	FIND_VALUE: Find a value in the network. A node will respond with the value of the key, if not present, it will act as FIND_NODE.
	STORE: Instruct a node to store a key-value pair.
*/
type Addr string
type Key string
type Value []Addr
type NodeId []byte

type RoutingTable struct {
	bucketLen int
	kBuckets  [][]NodeId
}
type Node struct {
	nodeId       NodeId
	kvStore      map[Key]Value
	routingTable *RoutingTable
}

func NewNode(id NodeId, bucketLen int) *Node {
	return &Node{
		nodeId:  id,
		kvStore: make(map[Key]Value),
		routingTable: &RoutingTable{
			bucketLen: bucketLen,
			kBuckets:  make([][]NodeId, 20),
		},
	}
}

// Node Functions
func (n1 *Node) operationPING(n2 *Node) {
	// a tcp dialup to n2 node.
}

func (n1 *Node) operationFINDNODE(n2 *Node) {
	// sends a tcp message to a node and will get back k nodes (ip:port)
}

func (n1 *Node) operationFINDVALUE(n2 *Node, k Key) {
	// sends a tcp message to a node an will either get back a value associated to that key, or //the same as FINDNODES
}

func (n1 *Node) operationSTORE(n2 *Node, k Key, v Value) {
	// sends a tcp message to a node instruction to store a KV pair
}

// Routing Table Functions
func (rt *RoutingTable) AddNode(nodeId, selfId NodeId) {
	// Adds a node to appropriate bucket
}

// Normal Functions
func xorDistance(a, b NodeId) int {
	dist := 0
	for i := range a {
		dist |= int(a[i] ^ b[i])
	}
	return dist
}
