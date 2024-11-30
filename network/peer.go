package network

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"sort"
	"sync"
	"time"

	"p2p/file"
)

type ConnectionRequest struct {
	addr     string
	response chan *PeerConnection
	err      chan error
}

type Piece struct {
	pieceIndex  int // -> index of the piece requested
	offset      int // -> byte offset within the piece
	blockLength int // -> length of requested piece's block
}

type ReqMessage struct {
	messageType MessageType // -> Piece req, bitfield, choke, unchoke, etc
	piece       *Piece      // -> piece info incase the messagetype is piece req
	other       interface{} // -> if messageType not piece req.
}

type PeerConfig struct {
	Host              string
	Port              int
	MaxConnections    int
	ConnectionTimeout time.Duration
	BlockSize         int
}

type PeerConnection struct { // all the details are of the connected peer
	conn        net.Conn
	addr        string
	status      PeerStatus
	peerDetails PeerDetails
	mu          sync.RWMutex
}

type PeerDetails struct {
	lastUnchoked time.Time
	uploadRate   float64
	downloadRate float64
	bitfield     []HavePiece
	haveQueue    chan Piece
}

type Peer struct {
	config      PeerConfig
	peerType    PeerType
	file        *file.File
	connections map[string]*PeerConnection
	mu          sync.RWMutex
	peerDetails PeerDetails

	done    chan struct{}
	pieces  chan []byte
	errorCh chan error

	connectionRequests chan ConnectionRequest
	pendingConnections map[string]ConnectionState
	activeConnections  int

	uploadHistory   []float64
	downloadHistory []float64
}

type PeerStatus int

const (
	StatusChoked PeerStatus = iota
	StatusUnchoked
	StatusInterested
	StatusNotInterested
)

type PeerType int

const (
	TypeSeeder PeerType = iota
	TypeLeecher
)

type ConnectionState int

const (
	ConnectionPending ConnectionState = iota
	ConnectionActive
	ConnectionClosed
)

type HavePiece byte

const (
	FalseHavePiece HavePiece = iota
	TrueHavePiece
)

type MessageType int

const (
	PieceRequest MessageType = iota
	Bitfield
	Choke
	Unchoke
	Handshake
)

const (
	maxPendingConnections = 50
	chokeInterval         = 10 * time.Second
	keepAliveInterval     = 1 * time.Minute
)

type PeerInterface interface {
	Start(context.Context) error
	Stop()
	ConnectToPeer(context.Context, string) (*PeerConnection, error)
	GetPeerStats() map[string]interface{}
}

var _ PeerInterface = (*Peer)(nil)

// exported functions
func New(cfg PeerConfig, f *file.File, peerType PeerType) *Peer {
	peer := &Peer{
		config:      cfg,
		peerType:    peerType,
		file:        f,
		connections: make(map[string]*PeerConnection),
		peerDetails: PeerDetails{
			bitfield:     make([]HavePiece, f.Pieces),
			haveQueue:    make(chan Piece, 10),
			uploadRate:   0,
			downloadRate: 0,
			lastUnchoked: time.Now(),
		},

		done:    make(chan struct{}),
		pieces:  make(chan []byte, f.Pieces),
		errorCh: make(chan error, 10),

		connectionRequests: make(chan ConnectionRequest),
		pendingConnections: make(map[string]ConnectionState),
		activeConnections:  0,

		uploadHistory:   make([]float64, 0, 10),
		downloadHistory: make([]float64, 0, 10),
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
		conn:   conn,
		addr:   req.addr,
		status: StatusChoked,
		peerDetails: PeerDetails{
			bitfield:     make([]HavePiece, p.file.Pieces),
			haveQueue:    make(chan Piece, 10),
			uploadRate:   0,
			downloadRate: 0,
			lastUnchoked: time.Now(),
		},
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
		if now.Sub(pConn.peerDetails.lastUnchoked) > p.config.ConnectionTimeout*2 {
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
		conn:   conn,
		addr:   conn.RemoteAddr().String(),
		status: StatusChoked,
		peerDetails: PeerDetails{
			bitfield:     make([]HavePiece, p.file.Pieces),
			haveQueue:    make(chan Piece, 10),
			uploadRate:   0,
			downloadRate: 0,
			lastUnchoked: time.Now(),
		},
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

			pc.peerDetails.lastUnchoked = time.Now()
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
		if pieceIndex < 0 || pieceIndex >= len(pc.peerDetails.bitfield) {
			return fmt.Errorf("invalid piece index")
		}
		exists := pc.peerDetails.bitfield[pieceIndex] == TrueHavePiece
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
		pc.status = StatusInterested
	case Bitfield:
		_, err := pc.conn.Write([]byte{byte(Bitfield)})
		if err != nil {
			return fmt.Errorf("failed to write bitfield: %w", err)
		}
		pc.status = StatusInterested
	}
	return nil
}

func (p *Peer) sendMessage(pc *PeerConnection, msg ReqMessage) error {
	var data []byte

	switch msg.messageType {
	case PieceRequest:
		if msg.piece == nil {
			return fmt.Errorf("missing piece info")
		}
		data = make([]byte, 5)
		data[0] = byte(PieceRequest)
		binary.BigEndian.PutUint32(data[1:], uint32(msg.piece.pieceIndex))
	case Bitfield:
		data = []byte{byte(Bitfield)}
	case Choke:
		data = []byte{byte(Choke)}
	case Unchoke:
		data = []byte{byte(Unchoke)}
	case Handshake:
		handshakeData, ok := msg.other.([]byte)
		if !ok {
			return fmt.Errorf("invalid handshke data")
		}
		data = make([]byte, 1+len(handshakeData))
		data[0] = byte(Handshake)
		copy(data[:1], handshakeData)
	default:
		return fmt.Errorf("unsuporrted message type")
	}

	msgLen := uint32(len(data))

	if err := binary.Write(pc.conn, binary.BigEndian, msgLen); err != nil {
		return fmt.Errorf("failed to write message length: %w", err)
	}

	if _, err := pc.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write message data: %w", err)
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

// choking algorithm
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
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.peerType {
	case TypeSeeder:
		/*
			Every 20 seconds:
			- Unchoke 3 peers based on:
				1. High download rates
				2. Latest unchoked time
			- Add 1 random peer (optimistic unchoke)
			- After 10 more seconds:
				- Choke the random peer
				- Unchoke the 4th highest peer in the ordered list
		*/
		var randomPeer *PeerConnection
		interestedPeers := make([]*PeerConnection, 0)
		for _, pc := range p.connections {
			if pc.status == StatusInterested {
				interestedPeers = append(interestedPeers, pc)
			}
		}

		sort.Slice(interestedPeers, func(i, j int) bool {
			return interestedPeers[i].peerDetails.downloadRate > interestedPeers[j].peerDetails.downloadRate
		})

		for i := 0; i < min(3, len(interestedPeers)); i++ {
			p.sendMessage(interestedPeers[i], ReqMessage{
				messageType: Unchoke,
			})
		}
		if len(interestedPeers) > 3 {
			randomPeer = interestedPeers[rand.IntN(len(interestedPeers))]
			p.sendMessage(randomPeer, ReqMessage{
				messageType: Unchoke,
			})
			time.AfterFunc(10*time.Second, func() {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.sendMessage(randomPeer, ReqMessage{
					messageType: Choke,
				})

				if len(interestedPeers) >= 4 {
					fourthPeer := interestedPeers[3]
					p.sendMessage(fourthPeer, ReqMessage{
						messageType: Unchoke,
					})
				}
			})
		}
	case TypeLeecher:
		/*
			Every 10 seconds:
			- Unchoke top 3 peers with highest upload rates
			- Add 1 random peer (optimistic unchoke)
			- Punishes free riders by prioritizing peers who contribute
		*/
		interestedPeers := make([]*PeerConnection, 0)
		for _, pc := range p.connections {
			if pc.status == StatusInterested {
				interestedPeers = append(interestedPeers, pc)
			}
		}

		sort.Slice(interestedPeers, func(i, j int) bool {
			return interestedPeers[i].peerDetails.uploadRate > interestedPeers[j].peerDetails.downloadRate
		})

		for i := 0; i < min(3, len(interestedPeers)); i++ {
			p.sendMessage(interestedPeers[i], ReqMessage{
				messageType: Unchoke,
			})
		}

		if len(interestedPeers) > 3 {
			randomPeer := interestedPeers[rand.IntN(len(interestedPeers))]
			p.sendMessage(randomPeer, ReqMessage{
				messageType: Unchoke,
			})
		}
	}
}

func (p *Peer) updateUploadRates(uploadBytes int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	duration := time.Since(p.peerDetails.lastUnchoked)
	uploadRate := float64(uploadBytes) / duration.Seconds()

	p.uploadHistory = append(p.uploadHistory, uploadRate)

	if len(p.uploadHistory) > 10 {
		p.uploadHistory = p.uploadHistory[1:]
	}

	p.peerDetails.uploadRate = calculateAvg(p.uploadHistory)
}

func (p *Peer) updateDownloadRates(downloadBytes int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	duration := time.Since(p.peerDetails.lastUnchoked)

	downloadRate := float64(downloadBytes) / duration.Seconds()

	p.downloadHistory = append(p.downloadHistory, downloadRate)

	if len(p.downloadHistory) > 10 {
		p.downloadHistory = p.downloadHistory[1:]
	}

	p.peerDetails.downloadRate = calculateAvg(p.downloadHistory)
}

func calculateAvg(rates []float64) float64 {
	if len(rates) == 0 {
		return 0
	}

	var sum float64
	for _, rate := range rates {
		sum += rate
	}

	return sum / float64(len(rates))
}

func (p *Peer) closeAllConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.conn.Close()
	}
	p.connections = make(map[string]*PeerConnection)
}

func (p *Peer) pieceSelection() *Piece {
	if p.connections == nil {
		return nil
	}

	// select a random piece if the number of peers is less than 4
	if len(p.peerDetails.haveQueue) < 4 {
		pieceIndex := rand.IntN(p.file.Pieces)
		if p.peerDetails.bitfield[pieceIndex] == FalseHavePiece {
			return &Piece{
				pieceIndex:  pieceIndex,
				offset:      0,
				blockLength: p.config.BlockSize,
			}
		}
	}

	frqMap := make(map[*Piece]int)
	for _, pc := range p.connections {
		rareP := <-pc.peerDetails.haveQueue
		if _, exist := frqMap[&rareP]; !exist {
			frqMap[&rareP] = 1
		} else {
			frqMap[&rareP]++
		}
	}
	// find the rarest piece
	var minFreq int = int(^uint(0) >> 1)
	var rarestPiece *Piece

	for piece, freq := range frqMap {
		if freq < minFreq {
			minFreq = freq
			rarestPiece = piece
		}
	}
	return rarestPiece
}

func (p *Peer) calculateAvailablePiecesPercentage() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	availablePieces := 0
	for _, pc := range p.connections {
		for _, piece := range pc.peerDetails.bitfield {
			if piece == TrueHavePiece {
				availablePieces++
			}
		}
	}

	return availablePieces * 100 / p.file.Pieces
}
