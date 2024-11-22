package connection

import (
	"fmt"
	"log"
	"net"
)

func StartTCPconnection() {
	listener, err := net.Listen("tcp", ":5555")
	if err != nil {
		log.Println("error listening to tcp connection", err)
	}
	defer listener.Close()
	fmt.Println("Listening on port: 5555")
	for {
		fmt.Println("Waiting for a connnection to accept")
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error listening to tcp connection", err)
		}
		fmt.Println("Connecion established!")
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error reading from connection:", err)
			continue
		}
		fmt.Printf("Received: %s\n", string(buffer[:n]))

		_, err = conn.Write([]byte("cool got it"))
		if err != nil {
			log.Println("Error writing to connection:", err)
			continue
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
