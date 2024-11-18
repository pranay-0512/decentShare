package cmd

import (
	"fmt"
	"sync"

	"github.com/pion/stun"
)

func GetIP(wg *sync.WaitGroup) {
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
