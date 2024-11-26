package network

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"p2p/file"
)

type BitField struct {
	bitfield []HavePiece
}

type PeerConfig struct {
	Host              string
	Port              int
	MaxConnections    int
	ConnectionTimeout time.Duration
	BlockSize         int
}

type Peer struct {
	config      PeerConfig
	peerType    PeerType
	file        *file.File
	connections map[string]*PeerConnection
	bitfield    []HavePiece
	status      map[string]PeerStatus
	mu          sync.RWMutex

	done    chan struct{}
	pieces  chan []byte // TODO - make it buffered, enqueing pieces as they are requested
	errorCh chan error

	connectionRequests chan ConnectionRequest
	pendingConnections map[string]ConnectionState
	activeConnections  int
}

type PeerConnection struct {
	conn     net.Conn
	addr     string
	status   PeerStatus
	lastSeen time.Time
}

type ConnectionRequest struct {
	addr     string
	response chan *PeerConnection
	err      chan error
}

type Piece struct {
	pieceIndex  int // -> index of the piece requested
	offset      int // -> byte offset within the piece
	blockLength int // -> length of requested piece
}

type ReqMessage struct {
	messageType MessageType // -> Piece req, bitfield, choke, unchoke, etc
	piece       *Piece      // -> piece info incase the messagetype is piece req
	other       interface{} // -> if messageType not piece req.
}

type PeerStatus int

type PeerType int

type ConnectionState int

type HavePiece byte

type MessageType int

const (
	PieceRequest MessageType = iota
	Bitfield
	Choke
	Unchoke
	Handshake
)

const (
	FalseHavePiece HavePiece = iota
	TrueHavePiece
)

const (
	StatusChoked PeerStatus = iota
	StatusUnchoked
	StatusInterested
	StatusNotInterested
)

const (
	TypeSeeder PeerType = iota
	TypeLeecher
)

const (
	maxPendingConnections = 50
	chokeInterval         = 10 * time.Second
	keepAliveInterval     = 2 * time.Minute
)

const (
	ConnectionPending ConnectionState = iota
	ConnectionActive
	ConnectionClosed
)

type PeerInterface interface {
	Start(context.Context) error
	Stop()
	ConnectToPeer(context.Context, string) (*PeerConnection, error)
	GetPeerStats() map[string]interface{}
}

var _ PeerInterface = (*Peer)(nil)

// exported functions
func NewPeer(cfg PeerConfig, f *file.File, peerType PeerType) *Peer {
	peer := &Peer{
		config:      cfg,
		file:        f,
		peerType:    peerType,
		connections: make(map[string]*PeerConnection),
		bitfield:    make([]HavePiece, f.Pieces),
		status:      make(map[string]PeerStatus),
		done:        make(chan struct{}),
		pieces:      make(chan []byte, f.Pieces),
		errorCh:     make(chan error, 10),

		connectionRequests: make(chan ConnectionRequest),
		pendingConnections: make(map[string]ConnectionState),
		activeConnections:  0,
	}
	return peer
}

func (p *Peer) Start(ctx context.Context) error {
	go p.connectionManager(ctx)

	if err := p.startListener(ctx); err != nil {
		return err
	}

	go p.chokeManager(ctx)

	return nil
}

func (p *Peer) Stop() {
	close(p.done)
	p.closeAllConnections()
}

func (p *Peer) ConnectToPeer(ctx context.Context, addr string) (*PeerConnection, error) {
	response := make(chan *PeerConnection, 1)
	errChan := make(chan error, 1)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p.connectionRequests <- ConnectionRequest{
		addr:     addr,
		response: response,
		err:      errChan,
	}:
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case conn := <-response:
			return conn, nil
		case err := <-errChan:
			return nil, err
		case <-time.After(p.config.ConnectionTimeout):
			return nil, fmt.Errorf("Connection request time out")
		}
	}
}

func (p *Peer) GetPeerStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := map[string]interface{}{
		"active_connections":  p.activeConnections,
		"pending_connections": len(p.pendingConnections),
		"max_connections":     p.config.MaxConnections,
	}

	connectionStates := make(map[string]string)
	for addr := range p.connections {
		state := "unknown"
		switch p.pendingConnections[addr] {
		case ConnectionPending:
			state = "pending"
		case ConnectionActive:
			state = "active"
		case ConnectionClosed:
			state = "closed"
		}
		connectionStates[addr] = state
	}

	stats["connection_state"] = connectionStates
	return stats
}

// internal functions
func (p *Peer) connectionManager(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-p.connectionRequests:
			go p.handleConnectionRequests(ctx, req)
		case <-ticker.C:
			go p.cleanupConnections()
		}
	}
}

func (p *Peer) handleConnectionRequests(ctx context.Context, req ConnectionRequest) {
	p.mu.Lock()
	if existing, exists := p.connections[req.addr]; exists {
		p.mu.Unlock()
		if existing != nil {
			req.response <- existing
		} else {
			req.err <- fmt.Errorf("connection already exists but is invalid")
		}
		return
	}

	if p.activeConnections >= p.config.MaxConnections {
		p.mu.Lock()
		req.err <- fmt.Errorf("connection limit reached (%d)", p.config.MaxConnections)
		p.mu.Unlock()
		return
	}

	p.pendingConnections[req.addr] = ConnectionPending
	p.mu.Unlock()

	conn, err := p.dialWithTimeout(ctx, req.addr)
	if err != nil {
		p.mu.Lock()
		delete(p.pendingConnections, req.addr)
		p.mu.Unlock()
		req.err <- fmt.Errorf("failed to establish connection %w", err)
		return
	}

	peerConn := &PeerConnection{
		conn:     conn,
		addr:     req.addr,
		status:   StatusChoked,
		lastSeen: time.Now(),
	}

	p.mu.Lock()
	p.connections[req.addr] = peerConn
	p.pendingConnections[req.addr] = ConnectionActive
	p.activeConnections++
	p.mu.Unlock()

	go p.readMessages(ctx, peerConn)
	go p.keepAlive(ctx, peerConn)

	req.response <- peerConn
}

func (p *Peer) cleanupConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	for addr, pConn := range p.connections {
		if now.Sub(pConn.lastSeen) > p.config.ConnectionTimeout*2 {
			log.Printf("Cleaning up inactive connections to %s", addr)
			pConn.conn.Close()
			delete(p.connections, addr)
			delete(p.pendingConnections, addr)
			p.activeConnections--
		}
	}
}

func (p *Peer) dialWithTimeout(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: p.config.ConnectionTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (p *Peer) startListener(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	go func() {
		defer listener.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					log.Printf("Failed to accept connection: %v", err)
					continue
				} else {
					go p.handleConnection(ctx, conn)
				}
			}
		}
	}()

	return nil
}

func (p *Peer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	peerConn := &PeerConnection{
		conn:     conn,
		addr:     conn.RemoteAddr().String(),
		status:   StatusChoked,
		lastSeen: time.Now(),
	}

	p.mu.Lock()
	p.connections[peerConn.addr] = peerConn
	p.mu.Unlock()

	go p.readMessages(ctx, peerConn)

	go p.keepAlive(ctx, peerConn)
}

func (p *Peer) readMessages(ctx context.Context, pc *PeerConnection) {
	buffer := make([]byte, p.config.BlockSize)
	msgChan := make(chan []byte, 10)
	msgErrChan := make(chan error, 10)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgChan:
				if err := p.handleMessage(pc, msg); err != nil {
					msgErrChan <- err
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := pc.conn.SetReadDeadline(time.Now().Add(p.config.ConnectionTimeout)); err != nil {
				log.Printf("Failed to set read deadline: %v", err)
				return
			}

			var msgLen uint32
			if err := binary.Read(pc.conn, binary.BigEndian, &msgLen); err != nil {
				log.Printf("Failed to read message length: %v", err)
				return
			}

			n, err := io.ReadFull(pc.conn, buffer[:msgLen])
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				return
			}

			select {
			case msgChan <- buffer[:n]:
			case <-ctx.Done():
				return
			}

			pc.lastSeen = time.Now()
		}
		select {
		case err := <-msgErrChan:
			if err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		default:
		}
	}
}

func (p *Peer) handleMessage(pc *PeerConnection, msg []byte) error {
	switch msgType := MessageType(msg[0]); msgType {
	case PieceRequest:
		pieceIndex := int(msg[1])
		if pieceIndex < 0 || pieceIndex >= len(p.bitfield) {
			return fmt.Errorf("invalid piece index")
		}
		exists := p.bitfield[pieceIndex] == TrueHavePiece
		if !exists {
			return fmt.Errorf("piece not available")
		}
		chunk, err := p.file.ReadChunk(pieceIndex)
		if err != nil {
			return fmt.Errorf("failed to read chunk: %w", err)
		}
		_, err = pc.conn.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}
	case Bitfield:
		_, err := pc.conn.Write([]byte{byte(Bitfield)})
		if err != nil {
			return fmt.Errorf("failed to write bitfield: %w", err)
		}
	}
	return nil
}

func (p *Peer) keepAlive(ctx context.Context, pc *PeerConnection) {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.sendKeepAlive(pc); err != nil {
				log.Printf("Failed to send keep-alive: %v", err)
				return
			}
		}
	}
}

func (p *Peer) sendKeepAlive(pc *PeerConnection) error {
	return binary.Write(pc.conn, binary.BigEndian, uint32(0))
}

func (p *Peer) chokeManager(ctx context.Context) {
	ticker := time.NewTicker(chokeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.updateChokes()
		}
	}
}

func (p *Peer) updateChokes() {
	// Implement optimistic unchoking algorithm
	// This would typically involve:
	// 1. Ranking peers by their upload/download rates
	// 2. Unchoking the top N peers
	// 3. Randomly unchoking one additional peer (optimistic unchoke)
	p.mu.Lock()
	defer p.mu.Unlock()
}

func (p *Peer) closeAllConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.conn.Close()
	}
	p.connections = make(map[string]*PeerConnection)
}
