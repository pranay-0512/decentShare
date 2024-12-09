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
	selfId   NodeId
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
	distance := xorDistance(rt.selfId, id)
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
		distI := xorDistance(target, contacts[i].ID)
		distJ := xorDistance(target, contacts[j].ID)
		return distI.Cmp(distJ) < 0
	})

	if len(contacts) < k {
		return contacts
	}

	return contacts[:k]
}

// node
func (n *Node) opsStore(key Key, value Value) error {
	n.storeMtx.Lock()
	defer n.storeMtx.Unlock()

	n.kvStore[key] = value
	return nil
}

func (n *Node) opsFindValue(key Key) (Value, bool) {
	n.storeMtx.RLock()
	defer n.storeMtx.RUnlock()

	value, exists := n.kvStore[key]
	return value, exists
}

func (n *Node) opsPing(contact *Contact) bool {
	//TODO implement the ping/pong logic
	_ = contact
	return false
}

func (n *Node) opsFindNode(target NodeId) []*Contact {
	return n.routingTable.FindClosestContacts(target, KBucketSize)
}

func generateNodeID(addr net.Addr) NodeId {
	hash := sha1.Sum([]byte(addr.String()))
	var nodeId NodeId
	copy(nodeId[:], hash[:])
	return nodeId
}

func xorDistance(a, b NodeId) *big.Int {
	aInt := new(big.Int).SetBytes(a[:])
	bInt := new(big.Int).SetBytes(b[:])

	xorDist := new(big.Int).Xor(aInt, bInt)
	return xorDist
}
