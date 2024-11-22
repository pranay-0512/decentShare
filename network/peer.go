package network

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"p2p/config"
	"p2p/file"
)

type Peer struct {
	config      config.NetworkConfig
	chunker     *file.FileChunker
	merger      *file.FileMerger
	mu          sync.Mutex
	connections map[string]net.Conn
}

func NewPeer(cfg config.NetworkConfig) *Peer {
	return &Peer{
		config:      cfg,
		chunker:     file.NewFileChunker(cfg.ChunkSize),
		merger:      file.NewFileMerger(),
		connections: make(map[string]net.Conn),
	}
}

func (p *Peer) Listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", p.config.Host, p.config.Port))
	if err != nil {
		return fmt.Errorf("error setting up TCP listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Peer listening on %s:%s", p.config.Host, p.config.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go p.handleIncomingConnection(conn)
	}
}

func (p *Peer) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Incoming connection from %s", remoteAddr)

	p.mu.Lock()
	p.connections[remoteAddr] = conn
	p.mu.Unlock()

	// Implement your incoming connection logic here
	// For example, receive file or handle peer discovery
}

func (p *Peer) DialAndSendFile(targetAddr, filename string) error {
	conn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", targetAddr, err)
	}
	defer conn.Close()

	p.mu.Lock()
	p.connections[targetAddr] = conn
	p.mu.Unlock()

	chunks, err := p.chunker.Chunkify(filename)
	if err != nil {
		return fmt.Errorf("failed to chunk file: %v", err)
	}

	log.Printf("Sending file %s to %s", filename, targetAddr)
	for i, chunk := range chunks {
		if _, err := conn.Write(chunk); err != nil {
			return fmt.Errorf("failed to send chunk %d: %v", i, err)
		}
	}

	return nil
}

func (p *Peer) ReceiveFile(outputFilename string) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", p.config.Host, p.config.Port))
	if err != nil {
		return fmt.Errorf("error setting up TCP listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Waiting for file transfer on %s:%s", p.config.Host, p.config.Port)

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("error accepting connection: %v", err)
	}
	defer conn.Close()

	var chunks [][]byte
	buf := make([]byte, p.config.ChunkSize)

	for {
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading chunk: %v", err)
		}

		chunk := make([]byte, bytesRead)
		copy(chunk, buf[:bytesRead])
		chunks = append(chunks, chunk)
	}

	if err := p.merger.Merge(outputFilename, chunks); err != nil {
		return fmt.Errorf("failed to merge received file: %v", err)
	}

	log.Printf("File received and saved as %s", outputFilename)
	return nil
}

func (p *Peer) CloseAllConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, conn := range p.connections {
		conn.Close()
		delete(p.connections, addr)
	}
}
