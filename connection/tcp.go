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
		defer conn.Close()
		fmt.Println("Connecion established!")
	}
}
