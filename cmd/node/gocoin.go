package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/blockchain"
	"os"
	"runtime/debug"
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
	cleanup()
	initDirs()

	bc, err := blockchain.NewBlockchain("/tmp/gocoin")
	mainAddress, err := bc.DiskWallet.NewAddress()

	for {
		b, _ := bc.Mine([]byte("coinbase"), mainAddress, blockchain.BLOCK_REWARD)

		err = bc.AddBlock(b)

		if err != nil {
			debug.PrintStack()
			panic(err)
		}

		err = bc.DiskWallet.ProcessBlock(b)

		if err != nil {
			debug.PrintStack()
			panic(err)
		}
	}
}

func initDirs() {
	err := os.Mkdir("/tmp/gocoin", 0777)
	err = os.Mkdir("/tmp/gocoin/data", 0777)
	err = os.Mkdir("/tmp/gocoin/db", 0777)

	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}

func cleanup() {
	err := os.RemoveAll("/tmp/gocoin")
	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}
