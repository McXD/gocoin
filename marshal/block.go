package marshal

import (
	"bytes"
	"gocoin/core"
)

const MAGIC_TX uint32 = 0xefefefef
const S_BLOCKHEADER = 80

var TX_SEP []byte

func init() {
	TX_SEP = Uint32ToBytes(MAGIC_TX)
}

func BlockHeader(bh *core.BlockHeader) []byte {
	var buf []byte

	buf = append(buf, bh.HashPrevBlock[:]...)      // PrevBlockHash, 32
	buf = append(buf, bh.HashMerkleRoot[:]...)     // MerkleRootHash, 32
	buf = append(buf, IntToBytes(int(bh.Time))...) // Time, 8
	buf = append(buf, Uint32ToBytes(bh.NBits)...)  // NBits, 4
	buf = append(buf, Uint32ToBytes(bh.Nonce)...)  // Nonce, 4

	return buf
}

func UBlockHeader(buf []byte) *core.BlockHeader {
	bh := &core.BlockHeader{
		Time:           0,
		NBits:          0,
		Nonce:          0,
		HashPrevBlock:  core.Hash256{},
		HashMerkleRoot: core.Hash256{},
	}

	p := 0
	bh.HashPrevBlock = core.Hash256FromSlice(buf[:32])

	p += 32
	bh.HashMerkleRoot = core.Hash256FromSlice(buf[p : p+32])

	p += 32
	bh.Time = int64(IntFromBytes(buf[p : p+8]))

	p += 8
	bh.NBits = Uint32FromBytes(buf[p : p+4])

	p += 4
	bh.Nonce = Uint32FromBytes(buf[p : p+4])

	return bh
}

func Block(block *core.Block) []byte {
	var buf []byte

	buf = append(buf, Uint32ToBytes(block.Height)...)         // Height, 4
	buf = append(buf, BlockHeader(&block.BlockHeader)...)     // Header, 80
	buf = append(buf, IntToBytes(len(block.Transactions))...) // Tx GetBlockFileSize, 8

	for _, tx := range block.Transactions {
		txSlice := Transaction(tx)
		buf = append(buf, txSlice...) // Tx, variable
		buf = append(buf, TX_SEP...)  // Separator, 4
	}

	return buf
}

func UBlock(buf []byte) *core.Block {
	block := &core.Block{
		Hash:   core.Hash256{},
		Height: 0,
		BlockHeader: core.BlockHeader{
			Time:           0,
			NBits:          0,
			Nonce:          0,
			HashPrevBlock:  core.Hash256{},
			HashMerkleRoot: core.Hash256{},
		},
		Transactions: []*core.Transaction{},
	}

	p := 0
	block.Height = Uint32FromBytes(buf[:4])

	p += 4
	block.BlockHeader = *UBlockHeader(buf[p : p+S_BLOCKHEADER])

	p += S_BLOCKHEADER
	txCount := IntFromBytes(buf[p : p+8])

	p += 8
	txs := bytes.Split(buf[p:], TX_SEP)
	for i := 0; i < txCount; i++ {
		block.Transactions = append(block.Transactions, UTransaction(txs[i]))
	}

	block.Hash = block.BlockHeader.Hash()
	return block
}
