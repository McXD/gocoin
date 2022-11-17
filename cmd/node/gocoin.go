package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/blockchain"
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
	// log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}

// run a gocoin node
func main() {
	greeting()

	blockchain, err := blockchain.NewBlockchain("/tmp/gocoin")
	if err != nil {
		panic(err)
	}

	for {
		b := blockchain.Mine([]byte("coinbase"), blockchain.Addresses[0], 1000)
		err := blockchain.AddBlock(b)
		if err != nil {
			panic(err)
		}
		blockchain.Wallet.ProcessBlock(b)
	}
}
