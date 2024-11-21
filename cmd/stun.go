package cmd

import (
	"net"

	"github.com/pion/stun"
)

func GetIP() (net.IP, int) {
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}

	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		panic(err)
	}
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var ip net.IP
	var port int
	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			panic(err)
		}
		ip = xorAddr.IP
		port = xorAddr.Port
	}); err != nil {
		panic(err)
	}
	return ip, port
}

// func GetIpTCP() {
// 	turnServer := ""
// 	turnUser := "test"
// 	turnPassword := "1234"
// 	realm := "testp2p.com"

// 	client, err := turn.NewClient(&turn.ClientConfig{
// 		TURNServerAddr: turnServer,
// 		Username:       turnUser,
// 		Password:       turnPassword,
// 		Realm:          realm,
// 	})
// 	if err != nil {
// 		log.Fatalf("Failed to create TURN client: %v", err)
// 	}

// }
