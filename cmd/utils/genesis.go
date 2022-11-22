package main

import (
	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
	"gocoin/blockchain"
	"gocoin/core"
)

func main() {
	coinbase := core.NewCoinBaseTransaction([]byte("genesis"), core.Hash160{}, blockchain.BLOCK_REWARD, 0)
	bb := core.NewBlockBuilder()
	bb.BaseOn(core.Hash256{}, 4294967295)
	bb.AddTransaction(coinbase)
	bb.NBits = blockchain.INITIAL_BITS
	bb.Time = blockchain.GENESIS_BLOCK_TIME

	if merkleRoot, err := bb.CalculateMerkleRoot(); err != nil {
		log.Warn(err)
	} else {
		bb.HashMerkleRoot = merkleRoot
	}

	target := bb.TargetValue()
	for {
		if bb.BlockHeader.Hash().Int().Cmp(target) == -1 {
			break
		}

		bb.Nonce++
	}

	bb.Hash = bb.BlockHeader.Hash()
	spew.Dump(bb.Block)

	//		Time: (int64) 1669004537,
	//		NBits: (uint32) 511705087,
	//		Nonce: (uint32) 219108,
	//		HashPrevBlock: (core.Hash256) (len=32 cap=32) 0000000000000000000000000000000000000000000000000000000000000000,
	//		HashMerkleRoot: (core.Hash256) (len=32 cap=32) 58A8C5C9B523F533A7F0403C0FDCFC1BFB77D68D3F84AEA36595D40026B86B0C
}
