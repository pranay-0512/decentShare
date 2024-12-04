package dht

import "net"

func DialTCP(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		panic(err)
	}
	return nil
}
