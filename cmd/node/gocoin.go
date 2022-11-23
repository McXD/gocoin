package main

import (
	"encoding/binary"
	"flag"
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

	// parse command line arguments
	mFlag := flag.Bool("m", true, "enable mining")
	rootFlag := flag.String("root", "/tmp/gocoin", "root directory")
	cFlag := flag.Bool("c", true, "clean up")
	p2pHostName := flag.String("p2p-host", "localhost", "p2p host name")
	p2pPort := flag.Int("p2p-port", 8844, "p2p port")
	rpcPort := flag.Int("rpc-port", 8080, "rpc port")
	seedFlag := flag.String("seed", "", "seed node multi-address")
	randSeed := flag.Int64("rand-seed", 0, "random seed")

	flag.Parse()

	if *cFlag {
		cleanup(*rootFlag)
		initDirs(*rootFlag)
	}

	bc, err := blockchain.NewBlockchain(*rootFlag, *p2pHostName, *p2pPort, *randSeed)
	shouldLog(err)
	if *cFlag {
		err = initWallet(bc.DiskWallet)
		shouldLog(err)
	}

	// start up servers
	go startRPC(*rpcPort, bc)
	go bc.StartP2PListener()

	// periodically discover peers
	if *seedFlag != "" {
		go bc.StartPeerDiscovery(*seedFlag)
	}

	if *mFlag {
		// start mining
		log.Infof("Start mining...")

		for {
			var timestamp [10]byte
			binary.PutVarint(timestamp[:], time.Now().UnixNano())
			coinbase := append(timestamp[:], []byte("coinbase")...)
			b, err := bc.Mine(coinbase, blockchain.BLOCK_REWARD)
			shouldLog(err)

			err = bc.ReceiveUnseenBlock(b)
			shouldLog(err)

			go bc.Network.BroadcastBlock(b)
		}
	} else {
		// periodically download blocks
		go bc.StartBlockDownloads()
	}

	time.Sleep(10000 * time.Second)
}

func startRPC(port int, bc *blockchain.Blockchain) {
	server := rpc.NewServer(port, bc)
	err := server.Run()
	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}

func initDirs(rootDir string) {
	err := os.Mkdir(rootDir, 0777)
	err = os.Mkdir(rootDir+"/data", 0777)
	err = os.Mkdir(rootDir+"/db", 0777)

	if err != nil {
		debug.PrintStack()
		panic(err)
	}
}

func initWallet(w *wallet.DiskWallet) error {
	// create 4 additional addresses
	for i := 0; i < 4; i++ {
		_, err := w.NewAddress()
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanup(rootDir string) {
	err := os.RemoveAll(rootDir)
	shouldLog(err)

}

func shouldLog(err error) {
	if err != nil {
		debug.PrintStack()
		log.Errorf("%v", err)
	}
}
