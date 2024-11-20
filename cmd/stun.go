package cmd

import "net"

// func GetIP() (net.IP, int) {
// 	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
// 	if err != nil {
// 		panic(err)
// 	}

// 	c, err := stun.DialURI(u, &stun.DialConfig{})
// 	if err != nil {
// 		panic(err)
// 	}
// 	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
// 	var ip net.IP
// 	var port int
// 	if err := c.Do(message, func(res stun.Event) {
// 		if res.Error != nil {
// 			panic(res.Error)
// 		}
// 		var xorAddr stun.XORMappedAddress
// 		if err := xorAddr.GetFrom(res.Message); err != nil {
// 			panic(err)
// 		}
// 		ip = xorAddr.IP
// 		port = xorAddr.Port
// 	}); err != nil {
// 		panic(err)
// 	}
// 	return ip, port
// }

func GetIP() (ip net.IP, i int) {
	// use turn server to get public ip
	return ip, i
}
