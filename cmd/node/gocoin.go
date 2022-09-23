package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/core"
	"os"
)

func greeting() {
	fmt.Print(`
  ________       _________        .__        
 /  _____/  ____ \_   ___ \  ____ |__| ____  
/   \  ___ /  _ \/    \  \/ /  _ \|  |/    \ 
\    \_\  (  <_> )     \___(  <_> )  |   |  \
 \______  /\____/ \______  /\____/|__|___|  /
        \/               \/               \/ 
`)
	println()
	println()
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

	bc := core.NewBlockchain()
	b1 := core.NewBlock(1, bc.Genesis.Hash, []byte{})
	bc.AddBlock(b1)

	b2 := core.NewBlock(2, b1.Hash, []byte{})
	bc.AddBlock(b2)

	b3 := core.NewBlock(3, b2.Hash, []byte{})
	bc.AddBlock(b3)

	fmt.Print(bc)
}
