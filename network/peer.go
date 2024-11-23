package network

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"p2p/config"
	"p2p/file"
)

type Peer struct {
	config          config.NetworkConfig
	file            *file.File
	connections     map[string]net.Conn
	interestedPeers map[bool]map[string]*Peer
	bitfield        []bool
	mu              sync.Mutex
}

const (
	MaxInterests     = 20
	MaxRegularChokes = 3
	MaxRandomChokes  = 1
	RunIntervalLeech = 10 * time.Second
	RunIntervalSeed  = 30 * time.Second
)

func NewPeer(cfg config.NetworkConfig, file *file.File) *Peer {
	return &Peer{
		config:          cfg,
		file:            file,
		connections:     make(map[string]net.Conn),
		interestedPeers: make(map[bool]map[string]*Peer),
		bitfield:        make([]bool, file.Pieces),
	}
}

func (p *Peer) ListenTCP() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", p.config.Host, p.config.Port))
	if err != nil {
		return fmt.Errorf("cannot listen on this port: %v", err)
	}
	defer listener.Close()

	log.Printf("Peer listening on %s:%s", p.config.Host, p.config.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		log.Printf("connected to: %s", conn.RemoteAddr().String())
		go p.handleIncomingConnection(conn)
	}
}

func (p *Peer) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	p.mu.Lock()
	p.connections[remoteAddr] = conn
	p.mu.Unlock()
	fmt.Println("connections of the peer: ", p.connections)
}

func (p *Peer) DialTCP(targetAddr string, file file.File) error {
	conn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", targetAddr, err)
	}
	defer conn.Close()

	p.mu.Lock()
	p.connections[targetAddr] = conn
	p.mu.Unlock()
	fmt.Println("connections of the peer: ", p.connections)
	return nil
}

func (p *Peer) SendPiece(targetAddr string, piece []byte) error {
	conn, ok := p.connections[targetAddr]
	if !ok {
		return fmt.Errorf("no connection to %s", targetAddr)
	}

	if _, err := conn.Write(piece); err != nil {
		return fmt.Errorf("failed to send piece %v", err)
	}
	return nil
}

func (p *Peer) ReceivePiece(conn net.Conn) ([]byte, error) {
	buf := make([]byte, p.config.BlockSize)
	bytesRead, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("error reading piece: %v", err)
	}

	piece := make([]byte, bytesRead)
	copy(piece, buf[:bytesRead])
	return piece, nil
}

func (p *Peer) CloseConn() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.Close()
	}
}

func (p *Peer) UnchokePeer(peer *Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interestedPeers[true][peer.config.Host] = peer
	delete(p.interestedPeers[false], peer.config.Host)
}

func (p *Peer) ChokePeer(peer *Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interestedPeers[false][peer.config.Host] = peer
	delete(p.interestedPeers[true], peer.config.Host)
}
