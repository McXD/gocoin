package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/rpc"
	"os"
)

func greeting() {
	fmt.Print(`
  __      _              __                       
 /__  _  /   _  o ._    (_ _|_  _. ._ _|_  _   _| 
 \_| (_) \_ (_) | | |   __) |_ (_| |   |_ (/_ (_| 


`)
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}

// run a gocoin node
func main() {
	greeting()

	var server = rpc.NewServer(8765)

	if err := server.Run(); err != nil {
		log.Warn("JSON-RPC server not started: %w", err)
	}

	//bc := core.NewBlockchain()
	//b1 := core.NewBlock(1, bc.Genesis.Hash, []byte{})
	//bc.AddBlock(b1)
	//
	//b2 := core.NewBlock(2, b1.Hash, []byte{})
	//bc.AddBlock(b2)
	//
	//b3 := core.NewBlock(3, b2.Hash, []byte{})
	//bc.AddBlock(b3)
	//
	//fmt.Print(bc)
}
