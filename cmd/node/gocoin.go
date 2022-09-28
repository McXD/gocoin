package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/core"
	"gocoin/internal/wallet"
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

	w := wallet.NewWallet()
	log.WithFields(log.Fields{
		"address": fmt.Sprintf("%X", w.Addresses[0]),
	}).Info("Created primary wallet with address.")

	bc := core.NewBlockchain(w.Addresses[0], 23, 1_000_000_000)
	log.WithFields(log.Fields{
		"genesisId":  bc.Head.Hash,
		"difficulty": bc.Head.Bits,
		"reward":     bc.Head.Reward,
	}).Info("Created blockchain")

	for {
		b := bc.Mine(w.Addresses[0])
		if err := bc.AddBlock(b); err != nil {
			log.Warn("error adding block: %s", err)
		}
		w.ProcessBlock(b)
	}
}
