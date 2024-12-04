package dht

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"
)

/*
Each machine acts as a node. Each node will have 2 things. A KV store and a routing table.
A KV store is a map of keys (file hash) and values (peer pool).
A routing table is a trie of nodes. Each node & key will be a representation of 160bit integer.
For example: 10101...
We make sure that a key is stored in the node that is closest to the key.
For this we will use XOR distance metric.

Routing table:
Each node will have 3 buckets, each buckets being the closest node to that node.
If the k-buckets are full, ping the least recently seen node. If it pongs, discard the new node. Else expel the least recently seen node and add the new node to the tail.

4 main operations:
PING: Check health of a node.const
FIND_NODE: Find a node in the network. A node will respond with the k closest nodes to the target node.
FIND_VALUE: Find a value in the network. A node will respond with the value of the key, if not present, it will act as FIND_NODE.
STORE: Instruct a node to store a key-value pair.

A node needs to know atleast 1 (/max k) nodes in each subtree that it is not a part of.
The question is how to implement the subtree wala thingy??
Since the DHT is not a physical datastructure, more so of a concept.
Each node(machine/peer) will hold information about atleast n other nodes (in total) present in subtrees that the node is not a part of. n is the n-bit id used.
Or to simplify, it will store nodes in buckets, and these buckets will be different based on the XOR range.
eg: [1,2), [2,4), [4,8), [8,16), [16,32), etc.

consider a 3bit id tree. N1 is 000, N2 is 011, N3 is 100, N4 is 110, N5 is 111, N6 is 010

					      Root
					/   		  \
				  0     		   1
				/       		    \
			  / \       		   / \
			0    1      		  0   1
		  /       \     		/      \
		/ \      / \          / \     / \
	  0	   1   0    1       0    1   0   1
	N1        N6    N2     N3       N4   N5

How will the routing table of node N2 look.

	{
		bucketLen = 3,
		kBuckets = {
			[
			[N6], --> [nodes with 1<=XOR<2] (closest to N2)
			[N1], --> [nodes with 2<=XOR<4]
			[N3, N4, N5], --> [nodes with 4<=XOR<8] (it will know about atleast 1 node amongst all of these)
			]
		}
	}

how will routing work?
-- Given a key, it will find the XOR between keyid and nodeid, then it will see in it's routing table which bucket to look at, and forward the req to those nodes (they will do the same if they do not have the key info)
*/
const (
	IDLength    = 160
	KBucketSize = 20
)

type Addr string               // string(<ip:port>)
type Key string                // file hash
type Value []Addr              // List of peer addresses
type NodeId [IDLength / 8]byte // SHA-1 Hashed id (160 bits)

type Contact struct {
	ID       NodeId
	Address  Addr
	LastSeen time.Time
}
type RoutingTable struct {
	selfId   NodeId               // k
	kBuckets [IDLength][]*Contact // k -> k closest nodes
	bktMutex *sync.RWMutex
}
type Node struct {
	id           NodeId
	kvStore      map[Key]Value
	routingTable *RoutingTable
	storeMtx     *sync.RWMutex
}

func NewNode(id NodeId) *Node {
	return &Node{
		id:      id,
		kvStore: make(map[Key]Value),
		routingTable: &RoutingTable{
			selfId:   id,
			kBuckets: [IDLength][]*Contact{},
		},
	}
}

func XORDistance(a, b NodeId) *big.Int {
	aInt := new(big.Int).SetBytes(a[:])
	bInt := new(big.Int).SetBytes(b[:])

	xorDist := new(big.Int).Xor(aInt, bInt)
	return xorDist
}

// routing table
func (rt *RoutingTable) AddContact(contact *Contact) error {
	rt.bktMutex.Lock()
	defer rt.bktMutex.Unlock()

	bktIndex := rt.bucketIndexForNode(contact.ID)
	bkt := rt.kBuckets[bktIndex]

	for i, existingContact := range bkt {
		if bytes.Equal(existingContact.ID[:], contact.ID[:]) {
			bkt = append(append(bkt[:i], bkt[i+1:]...), existingContact)
			rt.kBuckets[bktIndex] = bkt
			return nil
		}
	}

	if len(bkt) < KBucketSize {
		rt.kBuckets[bktIndex] = append(bkt, contact)
		return nil
	}

	oldestContact := bkt[0]
	_ = oldestContact
	//TODO implement ping mechanism
	return fmt.Errorf("k-bucket is full and oldest contact is still active")
}

func (rt *RoutingTable) bucketIndexForNode(id NodeId) int {
	distance := XORDistance(rt.selfId, id)
	return IDLength - 1 - int(distance.BitLen())
}

func (rt *RoutingTable) FindClosestContacts(target NodeId, k int) []*Contact {
	rt.bktMutex.RLock()
	defer rt.bktMutex.RUnlock()

	var contacts []*Contact

	for _, bucket := range rt.kBuckets {
		contacts = append(contacts, bucket...)
	}

	sort.Slice(contacts, func(i, j int) bool {
		distI := XORDistance(target, contacts[i].ID)
		distJ := XORDistance(target, contacts[j].ID)
		return distI.Cmp(distJ) < 0
	})

	if len(contacts) < k {
		return contacts
	}

	return contacts[:k]
}

func (n *Node) Store(key Key, value Value) error {
	n.storeMtx.Lock()
	defer n.storeMtx.Unlock()

	n.kvStore[key] = value
	return nil
}

func (n *Node) Lookup(key Key) (Value, bool) {
	n.storeMtx.RLock()
	defer n.storeMtx.RUnlock()

	value, exists := n.kvStore[key]
	return value, exists
}

func GenerateNodeID(addr net.Addr) NodeId {
	hash := sha1.Sum([]byte(addr.String()))
	var nodeId NodeId
	copy(nodeId[:], hash[:])
	return nodeId
}

func (n *Node) Ping(contact *Contact) bool {
	//TODO implement the ping/pong logic
	return false
}

func (n *Node) FindNode(target NodeId) []*Contact {
	return n.routingTable.FindClosestContacts(target, KBucketSize)
}
