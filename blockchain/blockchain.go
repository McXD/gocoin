package blockchain

import (
	"bufio"
	"container/list"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"gocoin/core"
	"gocoin/marshal"
	"gocoin/p2p"
	"gocoin/persistence"
	"gocoin/wallet"
	"golang.org/x/exp/slices"
	"io"
	"math/big"
	"sync"
	"time"
)

const (
	INITIAL_BITS        = 0x1e7fffff
	GENESIS_BLOCK_TIME  = 1669004537 // updated when deployed
	BLOCK_REWARD        = 1000
	S_BLOCK_QUEUE       = 100
	EXPECTED_BLOCK_TIME = 15 // seconds
	P_BITS_ADJUSTMENT   = 20 // blocks
	P_BLOCK_DOWNLOAD    = 60 // seconds
	P_PEER_DISCOVERY    = 60 // seconds
	CTX_ADDRESS         = "address"
	CTX_PREV_HASH       = "prev_hash"
	CTX_PREV_HEIGHT     = "height"
)

type Blockchain struct {
	RootDir                     string    // root directory of the blockchain data
	*wallet.DiskWallet                    // built-in persisted wallet
	*persistence.BlockFile                // current block file
	*persistence.BlockIndexRepo           // block index repository
	*persistence.ChainStateRepo           // chain state repository
	mempool                     list.List // transaction memory pool
	mempoolMutex                sync.Mutex
	*p2p.Network                              // peer-to-peer network
	branch                      []*core.Block // a possible new branch (orphanage)
	branchMutex                 sync.Mutex
	addBlockHandlers            []func(*core.Block)
	reorgHandlers               []func(*core.Block, []*core.UXTO)
	MiningCtx                   context.Context // context for mining
	MingCtxMutex                sync.Mutex
	blockQueue                  chan *core.Block
}

// NewBlockchain creates a new blockchain at path as root directory.
// This method does _not_ overwrite existing blockchain state.
// Genesis block is created hard-coded.
func NewBlockchain(rootDir string, hostname string, port int) (*Blockchain, error) {
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
	net, err := p2p.NewNetwork(hostname, port, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot create network: %w", err)
	}

	b := Blockchain{
		RootDir:        rootDir,
		DiskWallet:     w,
		BlockFile:      bf,
		BlockIndexRepo: bi,
		ChainStateRepo: cs,
		Network:        net,
	}

	b.MiningCtx = context.Background()

	// create genesis
	genesis := makeGenesisBlock()
	err = b.addBlockAsTip(genesis)
	if err != nil {
		return nil, fmt.Errorf("cannot add genesis block: %w", err)
	}

	// initialize wallet
	// add one address
	addr1, err := b.DiskWallet.NewAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	// register hooks
	b.RegisterAddBlockHandler(b.DiskWallet.ProcessBlock)
	b.RegisterReorgHandler(b.DiskWallet.RollBack)

	// set initial contexts
	b.MiningCtx = context.WithValue(b.MiningCtx, CTX_ADDRESS, addr1)
	b.MiningCtx = context.WithValue(b.MiningCtx, CTX_PREV_HASH, genesis.Hash)
	b.MiningCtx = context.WithValue(b.MiningCtx, CTX_PREV_HEIGHT, genesis.Height)

	// initialize the block queue
	b.blockQueue = make(chan *core.Block, S_BLOCK_QUEUE)

	return &b, nil
}

func makeGenesisBlock() *core.Block {
	// TODO: hardcode the genesis block, instead of build it (which may lead to non-deterministic genesis block)
	coinbase := core.NewCoinBaseTransaction([]byte("genesis"), core.Hash160{}, BLOCK_REWARD, 0)
	bb := core.NewBlockBuilder()
	bb.BaseOn(core.Hash256{}, 4294967295)
	bb.AddTransaction(coinbase)
	bb.NBits = INITIAL_BITS
	bb.Time = GENESIS_BLOCK_TIME
	bb.Nonce = 28980
	mkRoot, _ := bb.CalculateMerkleRoot()
	bb.HashMerkleRoot = mkRoot
	bb.Hash = bb.BlockHeader.Hash()

	return bb.Block
}

// Mine a block. Transaction selection is based on the following rules:
// 1. The block is max 1 MB in size
// 2. The block must contain at least one coinbase transaction
// 3. Transactions with higher fees are preferred
func (bc *Blockchain) Mine(coinbase []byte, reward uint32) (*core.Block, error) {
	// read mining parameters from context
	addr, ok := bc.MiningCtx.Value(CTX_ADDRESS).(core.Hash160)
	if !ok {
		return nil, fmt.Errorf("failed to get address from context")
	}
	prevHash, ok := bc.MiningCtx.Value(CTX_PREV_HASH).(core.Hash256)
	if !ok {
		return nil, fmt.Errorf("failed to get prev hash from context")
	}
	prevHeight, ok := bc.MiningCtx.Value(CTX_PREV_HEIGHT).(uint32)
	if !ok {
		return nil, fmt.Errorf("failed to get prev height from context")
	}

	bb := core.NewBlockBuilder()
	bb.BaseOn(prevHash, prevHeight)
	nBits, err := bc.GetNBitsAtHeight(prevHeight + 1)
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

		log.Infof("Selected transaction for mining from mempool: hash=%s, fee=%d", tx.Hash(), fee)

		if blkSize > 10*1024 { // size of a single block is less 10 KB
			break
		}
	}

	coinbaseTx := core.NewCoinBaseTransaction(coinbase, addr, reward, txFee)
	txs = append([]*core.Transaction{coinbaseTx}, txs...) // prepend coinbase transaction

	for _, tx := range txs {
		bb.AddTransaction(tx)
	}

	log.Infof("Start mining block: prevBlockHash=%s, prevHeight=%d, difficulty=%08x", bb.HashPrevBlock.String(), bb.Height-1, bb.NBits)
	b := bb.Build()
	log.Infof("***Mined a block***: hash: %s, height: %d, difficulty=%08x, prevBlockHash=%s", b.Hash.String(), b.Height, b.NBits, b.HashPrevBlock.String())

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

func (bc *Blockchain) AddBlockToQueue(b *core.Block) {
	log.Infof("Add block to queue: %s", b.Hash.String())
	bc.blockQueue <- b
}

// ProcessBlockQueue processes a block from the queue according to the following rules:
// 1. If the block references the current tip, add it to the chain (as the new tip)
// 2. If the block references a stale block or a block in the orphan pool, add it to the orphan pool (possible fork). Reorganize when necessary.
// 3. Else drop it
//
// Verification is only performed for the new tip. The verification for orphaned blocks will be performed during reorganization.
func (bc *Blockchain) ProcessBlockQueue() {
	for {
		block := <-bc.blockQueue
		log.Infof("Processing a block: hash=%s, height=%d, prevBlockHash=%s", block.Hash.String(), block.Height, block.HashPrevBlock.String())

		// check if it references the tip
		tipHash, err := bc.GetCurrentBlockHash()
		if err != nil {
			log.Errorf("failed to get current block hash: %s", err)
			continue
		}

		if block.HashPrevBlock == tipHash {
			// update blockchain tip
			if err = bc.addBlockAsTip(block); err != nil {
				log.Errorf("failed to add block as tip: %s", err)
			}
			continue
		}

		// check if it references a stale block
		_, err = bc.BlockIndexRepo.GetBlockIndexRecord(block.HashPrevBlock)
		if err != nil && err != persistence.ErrNotFound {
			log.Errorf("failed to get block index record: %s", err)
			continue
		}

		if err == nil {
			bc.branch = []*core.Block{}
			bc.branch = append(bc.branch, block)
			log.Infof("Orphan block %s at %d has a known parent %s at on active chain. Marked as possible new branch", block.Hash, block.Height, block.HashPrevBlock)
			continue
		}

		// check if it references a block in the orphan pool
		if err == persistence.ErrNotFound {
			// this block is a child of the last block in the branch
			if len(bc.branch) != 0 && bc.branch[len(bc.branch)-1].Hash == block.HashPrevBlock {
				bc.branch = append(bc.branch, block)
				log.Infof("Orphan block %s is the tip of new branch. Appended.", block.Hash)

				// if the blockchain tip is lower than the branch tip, reorganize
				rec, err := bc.GetBlockIndexRecord(tipHash)
				if err != nil {
					log.Errorf("failed to get block index record for tip %s: %s", tipHash, err)
					continue
				}
				if rec.Height < bc.branch[len(bc.branch)-1].Height {
					log.Infof("Fork detected. Reorganizing to new branch...")
					if err := bc.Reorganize(bc.branch); err != nil {
						log.Errorf("failed to reorganize: %s", err)
						continue
					}

					bc.branch = []*core.Block{}
				}
			} else {
				log.Infof("Orphan block %s is dropped", block.Hash)
			}
		}
	}
}

func (bc *Blockchain) VerifyBlock(block *core.Block) error {
	nBits, err := bc.GetNBitsAtHeight(block.Height)
	if err != nil {
		return fmt.Errorf("failed to get nBits for block %d: %w", block.Height, err)
	}
	if err := block.Verify(bc.ChainStateRepo, nBits, 500, BLOCK_REWARD); err != nil {
		return err
	}

	return nil
}

// addBlockAsTip add the block as the active tip. The block is verified against the current state.
func (bc *Blockchain) addBlockAsTip(block *core.Block) error {
	bc.MingCtxMutex.Lock()
	defer bc.MingCtxMutex.Unlock()

	prevBlockIndex, err := bc.GetBlockIndexRecord(block.HashPrevBlock)
	if err == persistence.ErrNotFound {
		if block.HashPrevBlock != core.EmptyHash256() {
			return fmt.Errorf("failed to get previous block %s: %w", block.HashPrevBlock, err)
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

	// verify height and prev block hash
	if block.Height != prevBlockIndex.Height+1 {
		return fmt.Errorf("invalid block height: expected %d, got %d", prevBlockIndex.Height+1, block.Height)
	} else {
		if block.Height != 0 { // not genesis block
			if block.HashPrevBlock != prevBlockIndex.Hash() {
				return fmt.Errorf("invalid block prev hash: expected %s, got %s", prevBlockIndex.Hash(), block.HashPrevBlock)
			}
		}
	}

	if err := bc.VerifyBlock(block); err != nil {
		return fmt.Errorf("failed to verify block %s: %w", block.Hash.String(), err)
	}

	// update chain state
	var spent []*core.UXTO
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

	// update mining context
	bc.MiningCtx = context.WithValue(bc.MiningCtx, CTX_PREV_HASH, block.Hash)
	bc.MiningCtx = context.WithValue(bc.MiningCtx, CTX_PREV_HEIGHT, block.Height)

	log.Infof("Blockchain tip changes to: %s, height=%d", block.Hash, block.Height)

	// call handlers
	for _, handler := range bc.addBlockHandlers {
		handler(block)
	}

	return nil
}

func (bc *Blockchain) GetNBitsAtHeight(height uint32) (uint32, error) {
	if height == 0 {
		return INITIAL_BITS, nil
	}

	brLast, err := bc.GetBlockIndexRecordOfHeight(height - 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get block index record of height %d: %w", height-1, err)
	}

	if height == 1 || height%P_BITS_ADJUSTMENT != 1 {
		return brLast.NBits, nil
	} else {
		brAgo, err := bc.GetBlockIndexRecordOfHeight(height - P_BITS_ADJUSTMENT)
		if err != nil {
			return 0, fmt.Errorf("failed to get block index record of height %d: %w", height-P_BITS_ADJUSTMENT, err)
		}

		duration := brLast.Time - brAgo.Time // in seconds
		tmp := big.Int{}
		tmp.Mul(brAgo.TargetValue(), big.NewInt(duration))
		newTarget := big.Int{}
		newTarget.Div(&tmp, big.NewInt(int64(P_BITS_ADJUSTMENT*EXPECTED_BLOCK_TIME)))

		return core.ParseNBits(&newTarget), nil
	}
}

// For each connection, read the header and delegates to the proper handler
func (bc *Blockchain) handleStream(s network.Stream) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "addr", p2p.FullMultiAddr(s.Conn().RemoteMultiaddr(), s.Conn().RemotePeer()))
	ctx = context.WithValue(ctx, "peerId", s.Conn().RemotePeer())

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	buf := make([]byte, p2p.S_HEADER)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading header: %s", err)
		return
	}
	h := p2p.ReceiveHeader(buf)

	switch h.Command {
	case p2p.CMD_GETADDR:
		handleGetAddr(ctx, bc, rw, h)
		break
	case p2p.CMD_GETBLOCKS:
		handleGetBlocks(ctx, bc, rw, h)
		break
	case p2p.CMD_GETDATA:
		handleGetData(ctx, bc, rw, h)
		break
	case p2p.CMD_TX:
		handleBroadcastTx(ctx, bc, rw, h)
		break
	case p2p.CMD_BLOCK:
		handleBroadcastBlock(ctx, bc, rw, h)
		break
	}

	err = s.Close()
	if err != nil {
		log.Errorf("Error closing stream: %s", err)
	}
}

func (bc *Blockchain) StartP2PListener() {
	bc.Network.StartListening(bc.handleStream)
	log.Infof("Node starts listening at %s", bc.Network.GetAddress())
}

func (bc *Blockchain) DownloadBlocks() {
	for {
		log.Infof("Downloading blocks...")

		peers := bc.ListPeerIDs()

		// what we know now
		var knowns []core.Hash256
		count := 10
		blkId, err := bc.GetCurrentBlockHash()
		if err != nil {
			log.Errorf("Error getting tip hash: %s", err)
		}
		for ; count > 0; count-- {
			knowns = append(knowns, blkId)

			// get its previous block
			rec, err := bc.BlockIndexRepo.GetBlockIndexRecord(blkId)
			if err != nil {
				log.Errorf("Error getting block index record: %s", err)
				break
			}

			if rec.Height == 0 {
				// reached genesis
				break
			} else {
				blkId = rec.HashPrevBlock
			}
		}

		// check who has the most inventory
		var invs []p2p.Inventory
		var targetPeer peer.ID
		for _, p := range peers {
			tmp := bc.Network.GetBlocks(p, knowns, core.EmptyHash256())
			if len(tmp) > len(invs) {
				invs = tmp
				targetPeer = p
			}
		}

		if len(invs) == 0 {
			time.Sleep(10 * time.Second)
			continue
		}

		// download blocks
		// TODO: handle forks
		blocks := bc.Network.DownloadBlocks(targetPeer, invs[1:])
		for _, b := range blocks {
			bc.AddBlockToQueue(b)
		}

		if len(blocks) == 0 {
			log.Infof("Found 0 new blocks. Blockchain is update-to-date.")
		} else {
			tip := blocks[len(blocks)-1]
			log.Infof("Downloaded %d. Tip is now at %s at height %d", len(blocks), tip.Hash, tip.Height)
		}

		time.Sleep(P_BLOCK_DOWNLOAD * time.Second)
	}
}

func (bc *Blockchain) StartPeerDiscovery(seed string) {
	log.Infof("Starting peer discovery...")
	if err := bc.Network.AddPeer(seed); err != nil {
		log.Errorf("Error adding seed peer: %s", err)
	}

	for {
		if err := bc.getPeersFrom(seed); err != nil {
			log.Errorf("Failed to get peers from %s: %s", seed, err)
		}

		time.Sleep(P_PEER_DISCOVERY * time.Second)
	}
}

func (bc *Blockchain) getPeersFrom(target string) error {
	maddr, err := ma.NewMultiaddr(target)
	if err != nil {
		return fmt.Errorf("failed to parse multiaddr: %w", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer info: %w", err)
	}

	fetched, err := bc.Network.GetAddr(info.ID)
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	for _, addr := range fetched {
		if slices.Contains(bc.ListKnownAddrs(), addr) {
			continue
		}

		err := bc.Network.AddPeer(addr)
		if err != nil {
			return fmt.Errorf("failed to add peer: %w", err)
		}

		err = bc.getPeersFrom(addr)
		if err != nil {
			return fmt.Errorf("failed to get peers from %s: %w", addr, err)
		}
	}

	return nil
}

// Reorganize the blockchain to the new active tip. The given blocks should be a series of new blocks of the longest chain.
// The first one in the list should be the branch point, and the last one should be the new tip.
// After reorganization, the mempool is cleared.
func (bc *Blockchain) Reorganize(blocks []*core.Block) error {
	// TODO: Mark the records as stale instead of deleting them
	// TODO: or else we will never find these blocks again
	tipHash, err := bc.GetCurrentBlockHash()
	if err != nil {
		return fmt.Errorf("failed to get current block hash: %w", err)
	}

	for {
		// for every block:
		// 1. remove the block index
		// 2. add back revs for that block (uxtos spent)
		// 3. delete uxtos generated
		tipRec, err := bc.GetBlockIndexRecord(tipHash)
		if err != nil {
			return fmt.Errorf("failed to get block index record of %s: %w", tipHash, err)
		}

		tipFile, err := persistence.NewBlockFile(bc.RootDir, tipRec.BlockFileID)
		if err != nil {
			return fmt.Errorf("failed to open block file %d: %w", tipRec.BlockFileID, err)
		}

		tipRev := tipFile.Revs[tipRec.Offset]
		tipBlk := tipFile.Blocks[tipRec.Offset]
		for _, u := range tipRev {
			err = bc.ChainStateRepo.PutUXTO(u)
			if err != nil {
				return fmt.Errorf("failed to put uxto: %w", err)
			}
		}

		generatedUXTOs := core.GenerateUXTOsFromBlock(tipBlk)
		for _, u := range generatedUXTOs {
			err = bc.ChainStateRepo.RemoveUXTO(u.TxId, u.N)
			if err != nil {
				return fmt.Errorf("failed to delete uxto: %w", err)
			}
		}

		for _, handler := range bc.reorgHandlers {
			handler(tipBlk, tipRev)
		}

		err = bc.BlockIndexRepo.DeleteBlockIndexRecord(tipHash)
		if err != nil {
			return fmt.Errorf("failed to delete block index record of %s: %w", tipHash, err)
		}

		tipHash = tipRec.HashPrevBlock
		if tipHash == blocks[0].HashPrevBlock {
			// break at branching height
			break
		}
	}

	err = bc.ChainStateRepo.SetCurrentBlockHash(blocks[0].HashPrevBlock)
	if err != nil {
		return fmt.Errorf("failed to set current block hash: %w", err)
	}

	for _, block := range blocks {
		err = bc.addBlockAsTip(block)
		if err != nil {
			return fmt.Errorf("failed to add block %s as tip: %w", block.Hash, err)
		}
	}

	return nil
}

func (bc *Blockchain) RegisterAddBlockHandler(handler func(*core.Block)) {
	bc.addBlockHandlers = append(bc.addBlockHandlers, handler)
}

func (bc *Blockchain) RegisterReorgHandler(handler func(*core.Block, []*core.UXTO)) {
	bc.reorgHandlers = append(bc.reorgHandlers, handler)
}
