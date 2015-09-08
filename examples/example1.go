package main

import (
	"github.com/bnsl/gosource/agentx"
	"log"
	"time"
)

var (
	oid agentx.OID = agentx.OID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
)

func main() {

	log.SetFlags(log.Lshortfile)

	c, err := agentx.ConnectAndServe("tcp", "static-host-00:705", &agentx.OID{0, 1, 2, 3, 4})
	if err != nil {
		log.Printf("error: %v", err)
	}

	f := agentx.NewOIDTree()
	var e agentx.Counter64 = 123
	f.NewEntry(agentx.OID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, e)

	res := c.Register(oid, f)
	_ = res

	time.Sleep(time.Second * 60)

	c.Close()
}
