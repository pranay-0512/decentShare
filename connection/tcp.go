package connection

import (
	"fmt"
	"log"
	"net"
)

func StartTCPconnection() {
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Println("error listening to tcp connection", err)
	}
	defer listener.Close()
	fmt.Println("Listening on port: 1234")
	for {
		fmt.Println("Waiting for a connnection to accept")
		conn, err := listener.Accept()
		if err != nil {
			log.Println("error listening to tcp connection", err)
		}
		defer conn.Close()
		fmt.Println("Connecion established!")
	}
}

func readAndWrite(conn net.Conn) {
	// 
}
