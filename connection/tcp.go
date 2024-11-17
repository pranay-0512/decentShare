package connection

import (
	"fmt"
	"sync"

	"github.com/pion/stun"
)

func StartTCPconnection() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go getSTUN(&wg)
	wg.Wait()

	// listener, err := net.Listen("tcp", "115.245.205.158")
	// if err != nil {
	// 	log.Println("error listening to tcp connection", err)
	// }
	// defer listener.Close()
	// fmt.Println("Listening on port: 6769")
	// for {
	// 	fmt.Println("Waiting for a connnection to accept")
	// 	conn, err := listener.Accept()
	// 	if err != nil {
	// 		log.Println("error listening to tcp connection", err)
	// 	}
	// 	defer conn.Close()
	// 	fmt.Println("Connecion established!")
	// }
}

func getSTUN(wg *sync.WaitGroup) {
	defer wg.Done()
	u, err := stun.ParseURI("stun:stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}

	c, err := stun.DialURI(u, &stun.DialConfig{})
	if err != nil {
		panic(err)
	}
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			panic(err)
		}
		fmt.Println("your IP is", xorAddr.IP)
	}); err != nil {
		panic(err)
	}
}
