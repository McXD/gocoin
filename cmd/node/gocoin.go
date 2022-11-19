package main

import (
	"encoding/binary"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/blockchain"
	"gocoin/rpc"
	"gocoin/wallet"
	"os"
	"runtime/debug"
	"time"
)

func greeting() {
	fmt.Print(`
                                                
                     ,@,,,,,(&                    
            .,,,*,#   &,,,,     ,,@,,*            
            ,,,,@@@     ,* @@    *,,,,            
             @,,,@     ,@@@&   @,,,,,             
             *,,,,,,,,@...,.*,,,,,,,,             
             .,,,,,,,,,,%@@,,,,,,,,,,             
             @,,,,,,,,,,,,,,,,,,,,,,,             
              ,,,,,,,,,,,,,,,,,,,,,,,#            
           @@ ,,,,,,,,,,,,,,,,,,,,,,,@/           
             @,,,,,,,,,,,,,,,,,,,,,,,.            
             .,,,,,,,,,,,,,,,,,,,,,,,.            
             &,,,,,,,,,,,,,,,,,,,,,,,(            
              ,,,,,,,,,,,,,,,,,,,,,,(             
               @,,,,,,,,,,,,,,,,,,,               
              ...@  %,,,,,,,,.@   .@              
                                        
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

	err = initWallet(bc.DiskWallet)
	if err != nil {
		debug.PrintStack()
		panic(err)
	}
	miningAddr := bc.DiskWallet.ListAddresses()[0]

	go startRPC(8080, bc)

	for {
		var timestamp [10]byte
		binary.PutVarint(timestamp[:], time.Now().UnixNano())
		coinbase := append(timestamp[:], []byte("coinbase")...)
		b, _ := bc.Mine(coinbase, miningAddr, blockchain.BLOCK_REWARD)

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

func startRPC(port int, bc *blockchain.Blockchain) {
	server := rpc.NewServer(port, bc)
	err := server.Run()
	if err != nil {
		debug.PrintStack()
		panic(err)
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

func initWallet(w *wallet.DiskWallet) error {
	// create 5 additional addresses
	for i := 0; i < 5; i++ {
		_, err := w.NewAddress()
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanup() {
	err := os.RemoveAll("/tmp/gocoin")
	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}
