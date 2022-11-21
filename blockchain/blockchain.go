package blockchain

import (
	"container/list"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gocoin/core"
	"gocoin/marshal"
	"gocoin/persistence"
	"gocoin/wallet"
	"math/big"
)

const (
	INITIAL_BITS        = 0x1e7fffff
	BLOCK_REWARD        = 1000
	EXPECTED_BLOCK_TIME = 7  // seconds
	N_BITS_ADJUSTMENT   = 20 // every 20 blocks
)

type Blockchain struct {
	RootDir string
	*wallet.DiskWallet
	*persistence.BlockFile
	*persistence.BlockIndexRepo
	*persistence.ChainStateRepo
	// transactions in mempool is gaurenteed be valid according to the current state, readily to be mined
	mempool list.List
}

// NewBlockchain creates a new blockchain at path as root directory.
// This method does _not_ overwrite existing blockchain state.
func NewBlockchain(rootDir string) (*Blockchain, error) {
	w, err := wallet.NewDiskWallet(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load or create wallet: %w", err)
	}
	bi, err := persistence.NewBlockIndexRepo(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create block index: %w", err)
	}
	cs, err := persistence.NewChainStateRepo(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot create chain state: %w", err)
	}
	bfId, err := bi.GetCurrentFileId()
	if err == persistence.ErrNotFound {
		bfId = 0
	} else if err != nil {
		return nil, fmt.Errorf("cannot get current block file id: %w", err)
	}
	bf, err := persistence.NewBlockFile(rootDir, bfId)

	b := Blockchain{
		RootDir:        rootDir,
		DiskWallet:     w,
		BlockFile:      bf,
		BlockIndexRepo: bi,
		ChainStateRepo: cs,
	}

	return &b, nil
}

// Mine a block. Transaction selection is based on the following rules:
// 1. The block is max 1 MB in size
// 2. The block must contain at least one coinbase transaction
// 3. Transactions with higher fees are preferred
func (bc *Blockchain) Mine(coinbase []byte, minerAddress core.Hash160, reward uint32) (*core.Block, error) {
	log.Debugf("Start preparing a block")

	currentBlockHash, err := bc.ChainStateRepo.GetCurrentBlockHash()
	if err == persistence.ErrNotFound { // first block
		currentBlockHash = core.EmptyHash256()
	} else if err != nil {
		return nil, fmt.Errorf("failed to get current block hash: %w", err)
	}
	log.Debugf("Prev block hash: %s", currentBlockHash.String())

	currentBlockIndex, err := bc.GetBlockIndexRecord(currentBlockHash)
	if err == persistence.ErrNotFound && currentBlockHash == core.EmptyHash256() { // genesis block
		currentBlockIndex = &persistence.BlockIndexRecord{
			BlockHeader: core.BlockHeader{
				NBits: INITIAL_BITS, // TODO: initial difficulty
			},
			Height: 4294967295, // -1
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get block %x: %w", currentBlockHash[:], err)
	}

	bb := core.NewBlockBuilder()
	bb.BaseOn(currentBlockHash, currentBlockIndex.Height)
	nBits, err := bc.GetNBitsFor(bb.Block)
	if err != nil {
		return nil, fmt.Errorf("failed to get nBits for block: %w", err)
	}
	bb.SetNBits(nBits)
	log.Debugf("Current difficulty: %064x", bb.TargetValue())
	// transaction selection
	var txFee uint32
	var blkSize int
	var txs []*core.Transaction
	for e := bc.mempool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*core.Transaction)
		txs = append(txs, tx)
		fee := tx.CalculateFee(bc.ChainStateRepo)
		txFee += fee
		blkSize += len(marshal.Transaction(tx))

		log.Debugf("Selected transaction for mining: hash=%s, fee=%d", tx.Hash(), fee)

		if blkSize > 10*1024 { // size of a single block is less 10 KB
			break
		}
	}

	coinbaseTx := core.NewCoinBaseTransaction(coinbase, minerAddress, reward, txFee)
	txs = append([]*core.Transaction{coinbaseTx}, txs...) // prepend coinbase transaction

	for _, tx := range txs {
		bb.AddTransaction(tx)
	}

	log.Debugf("Start mining block: prevBlockHash=%s, height=%d, difficulty=%08x", bb.HashPrevBlock.String(), bb.Height, bb.NBits)
	b := bb.Build()
	log.Infof("***Mined a block***: hash: %s, prevBlockHash=%s, height=%d, difficulty=%08x", b.Hash.String(), bb.HashPrevBlock.String(), bb.Height, bb.NBits)

	return b, nil
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

	if bc.mempool.Len() == 0 {
		bc.mempool.PushBack(tx)
		return nil
	}

	fee := tx.CalculateFee(bc.ChainStateRepo)
	for e := bc.mempool.Front(); e != nil; e = e.Next() {
		if fee > e.Value.(*core.Transaction).CalculateFee(bc.ChainStateRepo) {
			bc.mempool.InsertBefore(tx, e)
			log.Infof("Added transaction input mempool: %s", tx.Hash())
			break
		}
	}

	return nil
}

// AddBlock to the active tip. The block must be valid according to the current state.
func (bc *Blockchain) AddBlock(block *core.Block) error {
	prevBlockIndex, err := bc.GetBlockIndexRecord(block.HashPrevBlock)
	if err == persistence.ErrNotFound {
		if block.HashPrevBlock != core.EmptyHash256() {
			return fmt.Errorf("failed to get previous block %s: %w", prevBlockIndex.Hash(), err)
		} else { // genesis block
			prevBlockIndex = &persistence.BlockIndexRecord{
				BlockHeader: core.BlockHeader{
					NBits: INITIAL_BITS, // TODO: initial difficulty
				},
				Height: 4294967295, // overflow it to 0
			}
		}
	} else if err != nil {
		return fmt.Errorf("failed to get previous block %s: %w", prevBlockIndex.Hash(), err)
	}

	var spent []*core.UXTO
	nBits, err := bc.GetNBitsFor(block)
	if err != nil {
		return fmt.Errorf("failed to get nBits for block %d: %w", block.Height, err)
	}
	if err := block.Verify(bc.ChainStateRepo, nBits, 500, BLOCK_REWARD); err != nil {
		return fmt.Errorf("failed to verify block: %w", err)
	}

	// update chain state
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

		// clean the mempool (TODO: INEFFICIENT)
		for e := bc.mempool.Front(); e != nil; e = e.Next() {
			if e.Value.(*core.Transaction).Hash() == tx.Hash() {
				bc.mempool.Remove(e)
			}
		}
	}

	// open a new one if the current block file when full
	if bc.BlockFile.GetBlockFileSize() > 10*1024 { // 10 KB (TODO: parameter)
		if err := bc.BlockFile.Close(); err != nil {
			return fmt.Errorf("failed to close block file %d: %w", bc.BlockFile.Id, err)
		}

		if bc.BlockFile, err = persistence.NewBlockFile(bc.RootDir, bc.BlockFile.Id+1); err != nil {
			return fmt.Errorf("failed to open block file %d: %w", bc.BlockFile.Id+1, err)
		}

		if err := bc.BlockIndexRepo.PutCurrentFileId(bc.BlockFile.Id); err != nil {
			return fmt.Errorf("failed to update current block file id: %w", err)
		}
	}

	// save block and rev
	err = bc.BlockFile.WriteBlock(block, spent)
	if err != nil {
		return fmt.Errorf("failed to write block %x to file %d: %w", block.Hash, bc.BlockFile.Id, err)
	}

	// index the block
	err = bc.BlockIndexRepo.PutBlockIndexRecord(block.Hash, &persistence.BlockIndexRecord{
		BlockHeader: block.BlockHeader,
		Height:      block.Height,
		TxCount:     uint32(len(block.Transactions)),
		BlockFileID: bc.BlockFile.Id,
		Offset:      uint32(bc.BlockFile.GetBlockCount() - 1),
	})
	if err != nil {
		return fmt.Errorf("failed to save block index record: %w", err)
	}

	// index the transactions
	for i, tx := range block.Transactions {
		log.Debugf("Indexed transaction %s", tx.Hash())
		err = bc.BlockIndexRepo.PutTransactionRecord(tx.Hash(), &persistence.TransactionRecord{
			BlockFileID: bc.BlockFile.Id,
			BlockOffset: uint32(bc.BlockFile.GetBlockCount() - 1),
			TxOffset:    uint32(i),
		})
		if err != nil {
			return fmt.Errorf("failed to save transaction index record: %w", err)
		}
	}

	// update block file info
	err = bc.BlockIndexRepo.PutFileInfoRecord(bc.BlockFile.Id, &persistence.FileInfoRecord{
		BlockCount:    uint32(bc.BlockFile.GetBlockCount()),
		BlockFileSize: uint32(bc.BlockFile.GetBlockFileSize()),
		UndoFileSize:  uint32(bc.BlockFile.GetUndoFileSize()),
	})
	if err != nil {
		return fmt.Errorf("failed to save file info record: %w", err)
	}

	// update tip block
	if err = bc.SetCurrentBlockHash(block.Hash); err != nil {
		return fmt.Errorf("failed to update current block hash: %w", err)
	}

	log.Infof("Blockchain tip changes to: %s, height=%d", block.Hash, block.Height)
	return nil
}

func (bc *Blockchain) GetNBitsFor(block *core.Block) (uint32, error) {
	if block.Height == 0 {
		return INITIAL_BITS, nil
	}

	brLast, err := bc.GetBlockIndexRecordOfHeight(block.Height - 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get block index record of height %d: %w", block.Height-1, err)
	}

	if block.Height == 1 || block.Height%N_BITS_ADJUSTMENT != 1 {
		return brLast.NBits, nil
	} else {
		brAgo, err := bc.GetBlockIndexRecordOfHeight(block.Height - N_BITS_ADJUSTMENT)
		if err != nil {
			return 0, fmt.Errorf("failed to get block index record of height %d: %w", block.Height-N_BITS_ADJUSTMENT, err)
		}

		duration := brLast.Time - brAgo.Time // in seconds
		tmp := big.Int{}
		tmp.Mul(brAgo.TargetValue(), big.NewInt(duration))
		newTarget := big.Int{}
		newTarget.Div(&tmp, big.NewInt(int64(N_BITS_ADJUSTMENT*EXPECTED_BLOCK_TIME)))

		log.Infof("Adjusted Difficulty from %08x to %08x", brAgo.NBits, core.ParseNBits(&newTarget))
		return core.ParseNBits(&newTarget), nil
	}
}

// Reorganize the blockchain to the new active tip. The given blocks should be a series of new blocks of the longest chain.
// After reorganization, the mempool is cleared.
func (bc *Blockchain) Reorganize(blocks []*core.Block) error {
	return nil
}
