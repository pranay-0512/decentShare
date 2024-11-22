package connection

import (
	"fmt"
	"log"
	"net"
	"p2p/file"
)

var filename = "/main.text"

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

	_, err = conn.Write([]byte("Hello from a peer in same network"))
	if err != nil {
		fmt.Println("Error writing to peer: ", err)
		panic(err)
	}

	fmt.Println("Sent message to peer")
	defer conn.Close()
}
