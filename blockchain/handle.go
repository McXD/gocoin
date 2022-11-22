package blockchain

import (
	"bufio"
	"container/list"
	"context"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"gocoin/core"
	"gocoin/p2p"
	"gocoin/persistence"
	"io"
)

func handleGetAddr(ctx context.Context, bc *Blockchain, rw *bufio.ReadWriter, _ p2p.Header) {
	// add to our own neighbor list
	addr := ctx.Value("addr").(string)
	err := bc.Network.AddPeer(addr)
	if err != nil {
		log.Errorf("Error adding peer: %s", err)
		return
	}

	_, err = rw.Write(p2p.SendAddr(bc.Network.ListKnownAddrs()))
	if err != nil {
		log.Errorf("Error writing response: %s", err)
		return
	}

	err = rw.Flush()
	if err != nil {
		log.Errorf("Error flushing response: %s", err)
		return
	}
}

// GetBlocks requests block ids from the given peer
// L ---- getblocks ----> R
// L <--- inv blocks ---- R
func handleGetBlocks(ctx context.Context, bc *Blockchain, rw *bufio.ReadWriter, h p2p.Header) {
	log.Infof("HandleGetBlocks request from %s", ctx.Value("addr").(string))

	buf := make([]byte, h.SPayload)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading payload: %s", err)
		return
	}
	msg := p2p.ReceiveGetBlocks(buf)

	// find the most recent block we have in common with the peer
	// the block hashes is ordered from most recent to oldest
	var i int
	var rec *persistence.BlockIndexRecord
	for i = 0; i < len(msg.BlockHashes); i++ {
		rec, err = bc.BlockIndexRepo.GetBlockIndexRecord(msg.BlockHashes[i])
		if err == persistence.ErrNotFound {
			continue
		} else if err != nil {
			log.Errorf("Error getting block index record: %s", err)
			return
		} else {
			// i now points to the most recent block we have in common with the peer
			break
		}
	}

	if rec == nil {
		// we don't have any of the blocks the peer is asking for
		// TODO: not found
		// return closes the connection
		log.Infof("Peer %s is asking for blocks we don't have", ctx.Value("addr").(string))
		return
	}

	// send to requested endpoint
	// ordered from oldest to most recent
	tipHash, err := bc.GetCurrentBlockHash()
	if err != nil {
		log.Errorf("Error getting current block hash: %s", err)
		return
	}

	invs := make([]p2p.Inventory, 0)
	for {
		hash := rec.Hash()
		height := rec.Height
		invs = append(invs, p2p.Inventory{
			TypeId: p2p.INV_BLOCK,
			Hash:   hash,
		})

		if hash == msg.EndHash || hash == tipHash {
			break
		} else {
			// TODO: bulk query
			rec, err = bc.GetBlockIndexRecordOfHeight(height + 1)
			if err != nil {
				log.Errorf("Error getting block index record at height %d: %s", height+1, err)
				return
			}
		}
	}

	log.Infof("Sending invs of size %d", len(invs))
	_, err = rw.Write(p2p.SendInv(invs))
	if err != nil {
		log.Errorf("Error writing response: %s", err)
		return
	}

	err = rw.Flush()
	if err != nil {
		log.Errorf("Error flushing response: %s", err)
		return
	}

	log.Infof("Sent %d blocks to %s", len(invs), ctx.Value("addr").(string))
}

func handleGetData(ctx context.Context, bc *Blockchain, rw *bufio.ReadWriter, h p2p.Header) {
	buf := make([]byte, h.SPayload)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading payload: %s", err)
		return
	}
	invs := p2p.ReceiveInv(buf)

	switch invs[0].TypeId {
	case p2p.INV_TX:
		break
	case p2p.INV_BLOCK:
		sendBlocks(bc, rw, invs)
		break
	}
}

func sendBlocks(bc *Blockchain, rw *bufio.ReadWriter, invs []p2p.Inventory) {
	for _, inv := range invs {
		blockIndex, err := bc.BlockIndexRepo.GetBlockIndexRecord(inv.Hash)
		if err != nil {
			log.Errorf("Error getting block index: %s", err)
			return
		}

		bf, err := persistence.NewBlockFile(bc.RootDir, blockIndex.BlockFileID)
		if err != nil {
			log.Errorf("Error opening block file: %s", err)
			return
		}

		block := bf.Blocks[blockIndex.Offset]
		block.Height = blockIndex.Height
		block.Hash = blockIndex.Hash()

		_, err = rw.Write(p2p.SendBlock(block))
		if err != nil {
			log.Errorf("Error writing block: %s", err)
			return
		}
		err = rw.Flush()
		if err != nil {
			log.Errorf("Error flushing block: %s", err)
			return
		}

		log.Infof("Sent block %s of height %d", block.Hash, block.Height)
	}
}

func handleBroadcastBlock(ctx context.Context, bc *Blockchain, rw *bufio.ReadWriter, h p2p.Header) {
	// TODO: interrupt current mining
	// TODO: wait for reorg to complete
	bc.branchMutex.Lock()

	buf := make([]byte, h.SPayload)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading payload: %s", err)
		return
	}

	block := p2p.ReceiveBlock(buf)
	log.Infof("Received block %s at height %d", block.Hash, block.Height)

	// check if we already have the block
	_, err = bc.BlockIndexRepo.GetBlockIndexRecord(block.Hash)
	if err == nil {
		// we already have the block
		log.Infof("Already have block %s at height %d. Dropped.", block.Hash, block.Height)
		return
	} else if err != persistence.ErrNotFound {
		log.Errorf("Error getting block index record: %s", err)
		return
	} else {
		// we don't have the block
		// record it and broadcast

		// check if it references the tip
		tipHash, err := bc.GetCurrentBlockHash()
		if err != nil {
			log.Errorf("Error getting current block hash: %s", err)
			return
		}

		if block.HashPrevBlock != tipHash {
			// TODO: orphan blocks need to be verified with reference its height
			// TODO: currently we just _assume_ they are valid
			// TODO: in this way, a reorg might fail due to an invalid block after we roll back UXTOs
			// TODO: and the blockchain will be stale and needs a sync with other nodes
			log.Infof("Reiceived an orphan block %s of height %d", block.Hash, block.Height)
			if _, err := bc.BlockIndexRepo.GetBlockIndexRecord(block.HashPrevBlock); err == nil {
				bc.branch = []*core.Block{}
				bc.branch = append(bc.branch, block)
				log.Infof("Orphan block %s has a known parent %s on active chain. Marked as possible new branch", block.Hash, block.HashPrevBlock)
			} else if err == persistence.ErrNotFound {
				if bc.branch[len(bc.branch)-1].Hash == block.HashPrevBlock {
					bc.branch = append(bc.branch, block)
					log.Infof("Orphan block %s is the tip of new branch. Appended.", block.Hash)

					// if the blockchain tip is lower than the branch tip, reorganize
					rec, err := bc.GetBlockIndexRecord(tipHash)
					if err != nil {
						log.Errorf("Error getting blockchain tip index: %s", err)
						return
					}
					if rec.Height < bc.branch[len(bc.branch)-1].Height {
						log.Infof("Fork detected. Reorganizing to new branch")
						if err := bc.Reorganize(bc.branch); err != nil {
							log.Errorf("Error reorganizing: %s", err)
							return
						}
						bc.branch = []*core.Block{}
					}
				} else {
					log.Infof("Orphan block %s is dropped", block.Hash)
				}
			} else {
				log.Errorf("Error getting block index record: %s", err)
				return
			}

			// TODO: see if we should request missing blocks

		} else {
			// update blockchain tip
			if err = bc.AddBlockAsTip(block); err != nil {
				log.Errorf("Error adding block: %s", err)
				return
			}
		}

		// broadcast
		go bc.Network.BroadcastBlock(block, ctx.Value("peerId").(peer.ID))
	}

	bc.branchMutex.Unlock()
}

func handleBroadcastTx(ctx context.Context, bc *Blockchain, rw *bufio.ReadWriter, h p2p.Header) {
	bc.mempoolMutex.Lock()

	buf := make([]byte, h.SPayload)
	_, err := io.ReadFull(rw, buf)
	if err != nil {
		log.Errorf("Error reading payload: %s", err)
		return
	}

	tx := p2p.ReceiveTx(buf)
	log.Infof("Received tx %s", tx.Hash())

	var e *list.Element
	for e = bc.mempool.Front(); e != nil && e.Value.(*core.Transaction).Hash() != tx.Hash(); e = e.Next() {
	}
	if e == nil {
		// we don't have the tx
		// record it and broadcast

		err := bc.ReceiveTransaction(tx)
		if err != nil {
			log.Errorf("Error adding transaction: %s", err)
			return
		}

		go bc.Network.BroadcastTx(tx, ctx.Value("peerId").(peer.ID))
	}

	bc.mempoolMutex.Unlock()
}
