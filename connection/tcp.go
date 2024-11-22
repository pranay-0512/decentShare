package connection

import (
	"io"
	"log"
	"net"
	"p2p/file"
	"time"
)

var filename = "C:/Users/linkp/OneDrive/Desktop/decentShare/connection/main.txt"

// StartTCPConnection sets up a TCP server to send file chunks
func StartTCPConnection() {
	listener, err := net.Listen("tcp", ":5555")
	if err != nil {
		log.Fatalf("Error setting up TCP listener: %v", err)
	}
	defer listener.Close()

	log.Println("Server listening on :5555")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Connection established with %s", conn.RemoteAddr())

	chunks, err := file.Chunkify(filename)
	if err != nil {
		log.Printf("Error chunkifying file: %v", err)
		return
	}

	start := time.Now()
	for i, chunk := range chunks {
		n, err := conn.Write(chunk)
		if err != nil {
			log.Printf("Error sending chunk %d: %v", i, err)
			return
		}
		log.Printf("Sent chunk %d of size %d bytes", i, n)
	}
	duration := time.Since(start)
	log.Printf("File transfer completed in %v", duration)
}

// DialTCP sets up a TCP client to receive file chunks
func DialTCP() {
	conn, err := net.Dial("tcp", "192.168.1.166:5555")
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to server")

	var chunks [][]byte
	buf := make([]byte, file.ChunkSize)
	start := time.Now()

	for {
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				log.Println("All chunks received")
				break
			}
			log.Printf("Error reading from connection: %v", err)
			return
		}

		chunk := make([]byte, bytesRead)
		copy(chunk, buf[:bytesRead])
		chunks = append(chunks, chunk)
		log.Printf("Received chunk of size %d bytes", bytesRead)
	}
	duration := time.Since(start)
	log.Printf("File transfer completed in %v", duration)

	outputFile := "C:/Users/linkp/OneDrive/Desktop/decentShare/connection/received_file.txt"
	err = file.Merge(outputFile, chunks)
	if err != nil {
		log.Printf("Error merging file: %v", err)
		return
	}

	log.Printf("File successfully reconstructed as: %s", outputFile)
}
