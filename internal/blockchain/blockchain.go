package blockchain

import (
	"container/list"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/internal/core"
	"gocoin/internal/persistence"
	"gocoin/internal/persistence/binary"
	"gocoin/internal/wallet"
)

type Blockchain struct {
	rootDir string
	*wallet.Wallet
	*persistence.BlockIndexRepo
	*persistence.ChainStateRepo
	// transactions in mempool is gaurenteed be valid according to the current state, readily to be mined
	mempool list.List
}

// LoadBlockchainFrom the given root directory, whose data includes:
// 1. block index
// 2. chain state
// 3. wallet
func LoadBlockchainFrom(rootDir string) (*Blockchain, error) {
	return nil, nil
}

// NewBlockchain creates a new blockchain at path as root directory.
func NewBlockchain(rootDir string) (*Blockchain, error) {
	w := wallet.NewWallet()
	bi, err := persistence.NewBlockIndexRepo(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create block index: %w", err)
	}
	cs, err := persistence.NewChainStateRepo(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create chain state: %w", err)
	}

	b := Blockchain{
		rootDir:        rootDir,
		Wallet:         w,
		BlockIndexRepo: bi,
		ChainStateRepo: cs,
	}

	return &b, nil
}

// Mine a block. Transaction selection is based on the following rules:
// 1. The block is around 1 MB in size
// 2. The block must contain at least one coinbase transaction
// 3. Transactions with higher fees are preferred
func (bc *Blockchain) Mine(coinbase []byte, minerAddress core.Hash160, reward uint32) *core.Block {
	currentBlockHash, err := bc.GetCurrentBlockHash()
	if err == persistence.ErrNotFound { // not found is likely to be the first block
		currentBlockHash = core.EmptyHash256()
	} else if err != nil {
		panic(fmt.Errorf("failed to get current block hash: %w", err))
	}

	currentBlock, err := bc.GetBlockIndexRecord(currentBlockHash)
	if err != nil && currentBlockHash != core.EmptyHash256() {
		panic(fmt.Errorf("failed to get block %x: %w", currentBlockHash[:], err))
	} else {
		currentBlock = &persistence.BlockIndexRecord{
			BlockHeader: core.BlockHeader{
				Bits: 20, // TODO: initial difficulty
			},
			Height: 4294967295, // overflow it to 0
		}
	}

	bb := core.NewBlockBuilder()
	bb.BaseOn(currentBlockHash, currentBlock.Height)
	bb.SetBits(currentBlock.Bits) // TODO: adjust difficulty

	// transaction selection
	var txFee uint32
	var blkSize int
	var txs []*core.Transaction
	for e := bc.mempool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*core.Transaction)
		txs = append(txs, tx)
		txFee += tx.CalculateFee(bc.ChainStateRepo)
		blkSize += len(binary.SerializeTransaction(tx))

		if blkSize > 1024*1024 { // 1 MB
			break
		}
	}

	coinbaseTx := core.NewCoinBaseTransaction(coinbase, minerAddress, reward, txFee)
	txs = append([]*core.Transaction{coinbaseTx}, txs...) // prepend coinbase transaction

	for _, tx := range txs {
		bb.AddTransaction(tx)
	}

	log.Infof("Start mining a block, prevBlockHash: %s", bb.HashPrevBlock)
	b := bb.Build()
	log.Infof("Mined a block: %s", b.Hash)

	return b
}

// ReceiveTransaction adds a transaction to the mempool according to the following rules:
// 1. The transaction must be valid according to the current state
// 2. The transaction must not repeat an existing transaction in the pool (i.e., spending the same UTXOs) (TODO: the operation is slow with this data structure)
// 3. Transactions in the pool are sorted by fee
func (bc *Blockchain) ReceiveTransaction(tx *core.Transaction) error {
	// TODO: can wrap the transaction to include more data, such as the fee, to be more efficient

	if err := tx.Verify(bc.ChainStateRepo); err != nil {
		return fmt.Errorf("failed to verify transaction: %w", err)
	}

	for e := bc.mempool.Front(); e != nil; e = e.Next() {
		if tx.Hash() == e.Value.(*core.Transaction).Hash() {
			return fmt.Errorf("transaction already exists in the mempool")
		}

		// TODO: check duplicated UXTOs
	}

	fee := tx.CalculateFee(bc.ChainStateRepo)
	for e := bc.mempool.Front(); e != nil; e = e.Next() {
		if fee > e.Value.(*core.Transaction).CalculateFee(bc.ChainStateRepo) {
			bc.mempool.InsertBefore(tx, e)
			break
		}
	}

	return nil
}

// AddBlock to the active tip. The block must be valid according to the current state.
func (bc *Blockchain) AddBlock(block *core.Block) error {
	prevBlock, err := bc.GetBlockIndexRecord(block.HashPrevBlock)
	if err != nil && block.HashPrevBlock != core.EmptyHash256() { // skip genesis block
		return fmt.Errorf("failed to get previous block %s: %w", prevBlock.Hash(), err)
	} else {
		prevBlock = &persistence.BlockIndexRecord{
			BlockHeader: core.BlockHeader{
				Bits: 20, // TODO: initial difficulty
			},
			Height: 4294967295, // overflow it to 0
		}

		if err := bc.PutCurrentFileId(0); err != nil { // initialize
			return fmt.Errorf("failed to put current file id: %w", err)
		}
	}

	var spent []*core.UXTO
	if err := block.Verify(bc.ChainStateRepo, prevBlock.Bits, 500, 1000); err != nil { //TODO: reward parameter
		return fmt.Errorf("failed to verify block: %w", err)
	}

	for _, tx := range block.Transactions {
		// process spent UXTOs
		for _, input := range tx.Ins {
			if tx.IsCoinbaseTx() {
				break
			}
			spent = append(spent, bc.GetUXTO(input.PrevTxId, input.N))

			// remove spent from chain state
			err := bc.RemoveUXTO(input.PrevTxId, input.N)
			if err != nil {
				return fmt.Errorf("failed to remove spent UXTO: %w", err)
			}
		}

		// record newly created UXTOs
		for o, output := range tx.Outs {
			err := bc.PutUXTO(&core.UXTO{
				TxId:  tx.Hash(),
				N:     uint32(o),
				TxOut: output,
			})

			if err != nil {
				return fmt.Errorf("failed to put UXTO: %w", err)
			}
		}

		// clear the mempool (TODO: INEFFICIENT)
		for e := bc.mempool.Front(); e != nil; e = e.Next() {
			if e.Value.(*core.Transaction).Hash() == tx.Hash() {
				bc.mempool.Remove(e)
			}
		}
	}

	// save block and rev
	fileId, err := bc.BlockIndexRepo.GetCurrentFileId()
	if err != nil {
		return fmt.Errorf("failed to get current block file id: %w", err)
	}

	blockFile, err := persistence.OpenBlockFile(bc.rootDir, fileId)
	if err != nil {
		return fmt.Errorf("failed to open block file %d: %w", fileId, err)
	}
	if blockFile.Size() > 200*1024*1024 { // 200MB (TODO: parameter)
		blockFile, err = persistence.OpenBlockFile(bc.rootDir, fileId+1)
		if err != nil {
			return fmt.Errorf("failed to open block file %d: %w", fileId+1, err)
		}
	}

	err = blockFile.WriteBlock(block, spent)
	if err != nil {
		return fmt.Errorf("failed to write block %x to file %d: %w", block.Hash, fileId, err)
	}

	// save block index
	err = bc.BlockIndexRepo.PutBlockIndexRecord(block.Hash, &persistence.BlockIndexRecord{
		BlockHeader: block.BlockHeader,
		Height:      prevBlock.Height + 1,
		TxCount:     uint32(len(block.Transactions)),
		BlockFileID: fileId + 1,
		Offset:      uint32(blockFile.GetBlockSize() - 1),
	})
	if err != nil {
		return fmt.Errorf("failed to save block index record: %w", err)
	}

	// update tip block
	if err = bc.SetCurrentBlockHash(block.Hash); err != nil {
		return fmt.Errorf("failed to update current block hash: %w", err)
	}

	log.Infof("Blockchain tip changes to: %s", block.Hash)

	blockFile.Close()

	return nil
}

// Reorganize the blockchain to the new active tip. The given blocks should be a series of new blocks of the longest chain.
// After reorganization, the mempool is cleared.
func (bc *Blockchain) Reorganize(blocks []*core.Block) error {
	return nil
}

func (bc *Blockchain) StartMine() {

}

func (bc *Blockchain) StartP2P() {

}

func (bc *Blockchain) StartRPC() {

}
