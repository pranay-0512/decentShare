package connection

import (
	"fmt"
	"io"
	"log"
	"net"
	"p2p/file"
)

var filename = "C:/Users/linkp/OneDrive/Desktop/decentShare/connection/main.txt"

func StartTCPconnection() {
	listener, err := net.Listen("tcp", ":5555")
	if err != nil {
		log.Println("error listening to tcp connection", err)
		return
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error listening to tcp connection", err)
		}
		chunks, err := file.Chunkify(filename)
		if err != nil {
			log.Println("cannot chunkify: ", err)
		}
		for _, chunk := range chunks {
			_, err := conn.Write(chunk)
			fmt.Println("writing this chunk: ", chunk)
			if err != nil {
				log.Println("cannot write the chunks to connection:", err)
			}
			defer conn.Close()
		}
	}
}

func DialTCP() {
	conn, err := net.Dial("tcp", "192.168.1.166:5555")
	if err != nil {
		fmt.Println("Error connection to peer: ", err)
		panic(err)
	}

	fmt.Println("Sent connection req to listening peer")

	var chunks [][]byte
	buf := make([]byte, file.ChunkSize)

	for {
		bytesRead, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("All chunks received")
				break
			}
			log.Println("Error reading from connection: ", err)
			return
		}

		// Store the received chunk
		chunk := make([]byte, bytesRead)
		copy(chunk, buf[:bytesRead])
		chunks = append(chunks, chunk)

		fmt.Printf("Received chunk of size %d bytes\n", bytesRead)
	}

	defer conn.Close()
	outputFile := "received_file.txt" // Destination file name
	err = file.Merge(outputFile, chunks)
	if err != nil {
		log.Println("Error merging chunks into file: ", err)
		return
	}

	fmt.Println("File successfully reconstructed as:", outputFile)
}
