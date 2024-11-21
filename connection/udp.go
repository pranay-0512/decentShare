package connection

import (
	"fmt"
	"net"
)

func Udp() {
	addr := net.UDPAddr{
		Port: 12345,
		IP:   net.ParseIP("180.151.192.116"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Listening for UDP data on port 12345...")
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		fmt.Printf("Received: %s from %s\n", string(buf[:n]), remoteAddr)
	}
}
