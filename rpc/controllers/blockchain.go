package controllers

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gocoin/blockchain"
	"gocoin/core"
	"gocoin/marshal"
	"gocoin/persistence"
	"net/http"
)

type BlockchainController struct {
	*blockchain.Blockchain
}

type sendFromForm struct {
	From   string `json:"from" binding:"required"`
	To     string `json:"to" binding:"required"`
	Amount uint32 `json:"amount" binding:"required"`
	Fee    uint32 `json:"fee" binding:"required"`
}

type TxInDTO struct {
	PrevTxid  string
	Vout      uint32
	ScriptSig string // encoded bytes for coinbase or ScripSig
}

type TxOutDTO struct {
	Address string
	Amount  uint32
}

type TransactionDTO struct {
	TxId    string
	Inputs  []TxInDTO
	Outputs []TxOutDTO
}

func (b *BlockchainController) GetTransaction(c *gin.Context) {
	txId, err := core.ParseHash256(c.Query("txId"))
	if err != nil {
		SendError(c, http.StatusBadRequest, err)
		return
	}

	txRecord, err := b.GetTransactionRecord(txId)
	if err != nil {
		SendError(c, http.StatusInternalServerError, err)
		return
	}

	tx, err := persistence.GetTransaction(b.Blockchain.RootDir, txRecord)

	if err != nil {
		SendError(c, http.StatusInternalServerError, err)
		return
	}

	txInDTOs := make([]TxInDTO, len(tx.Ins))
	for i, txIn := range tx.Ins {
		var raw []byte
		if tx.IsCoinbaseTx() {
			raw = txIn.Coinbase
		} else {
			raw = marshal.SerializeScriptSig(&txIn.ScriptSig)
		}

		txInDTOs[i] = TxInDTO{
			PrevTxid:  txIn.PrevTxId.String(),
			Vout:      txIn.N,
			ScriptSig: base64.StdEncoding.EncodeToString(raw),
		}
	}

	txOutDTOs := make([]TxOutDTO, len(tx.Outs))
	for i, txOut := range tx.Outs {
		txOutDTOs[i] = TxOutDTO{
			Address: txOut.PubKeyHash.String(),
			Amount:  txOut.Value,
		}
	}

	c.JSON(http.StatusOK, TransactionDTO{
		TxId:    tx.Hash().String(),
		Inputs:  txInDTOs,
		Outputs: txOutDTOs,
	})
}

// SendFrom sends an amount from the given address to the given address.
// From implementation's perspective, this is blockchain-level concern; but from client's perspective, this is wallet-level concern.
// So, this method is implemented in the blockchain controller and exposed as a wallet controller.
//
// POST /wallet/sendFrom
//
//	{
//		"from": "1JwSSubhmg6iPtRjtyqhUYYH7bZg3Lfy1T",
//		"to": "1FQc5LdgGHMHEN9nwkjmz6tWkxhPpxBvBU",
//		"amount": 1000,
//	 "fee": 50
//	}
func (b *BlockchainController) SendFrom(c *gin.Context) {
	var form sendFromForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	fromAddr := core.Hash160{}
	err := fromAddr.ParseAddress(form.From)
	if err != nil {
		SendError(c, http.StatusBadRequest, err)
		return
	}

	toAddr := core.Hash160{}
	err = toAddr.ParseAddress(form.To)
	if err != nil {
		SendError(c, http.StatusBadRequest, err)
		return
	}

	transaction, err := b.DiskWallet.CreateTransaction(fromAddr, toAddr, form.Amount, form.Fee)
	if err != nil {
		SendError(c, http.StatusInternalServerError, err)
		return
	}

	err = b.ReceiveTransaction(transaction)
	if err != nil {
		SendError(c, http.StatusInternalServerError, err)
		return
	}

	// broadcast
	go b.BroadcastTx(transaction)

	c.JSON(http.StatusOK, gin.H{
		"txId": transaction.Hash().String(),
	})
}

type MiningCtxDTO struct {
	MinerAddress string `json:"minerAddress"`
	PrevHash     string `json:"prevHash"`
}

// GetMiningContext returns the mining context.
// GET /blockchain/miningContext
func (b *BlockchainController) GetMiningContext(c *gin.Context) {
	b.MingCtxMutex.Lock()
	addr, ok1 := b.MiningCtx.Value(blockchain.CTX_ADDRESS).(core.Hash160)
	prevHash, ok2 := b.MiningCtx.Value(blockchain.CTX_PREV_HASH).(core.Hash256)
	b.MingCtxMutex.Unlock()

	if !ok1 || !ok2 {
		SendError(c, http.StatusInternalServerError, errors.New("invalid mining context"))
		return
	}

	c.JSON(http.StatusOK, MiningCtxDTO{
		MinerAddress: addr.String(),
		PrevHash:     prevHash.String(),
	})
}

// SetMiningContext sets the mining context (for next block mined)
// POST /blockchain/miningContext
func (b *BlockchainController) SetMiningContext(c *gin.Context) {
	var form MiningCtxDTO
	if err := c.ShouldBindJSON(&form); err != nil {
		SendError(c, http.StatusBadRequest, err)
		return
	}

	// async call
	go func() {
		b.MingCtxMutex.Lock()
		defer b.MingCtxMutex.Unlock()
		if form.MinerAddress != "" {
			addr := core.Hash160{}
			err := addr.ParseAddress(form.MinerAddress)
			if err != nil {
				SendError(c, http.StatusBadRequest, err)
			}
			b.MiningCtx = context.WithValue(b.MiningCtx, blockchain.CTX_ADDRESS, addr)
			log.Infof("mining context: miner address set to %s", addr.String())
		}

		if form.PrevHash != "" {
			prevHash, err := core.ParseHash256(form.PrevHash)
			if err != nil {
				SendError(c, http.StatusBadRequest, err)
			}
			b.MiningCtx = context.WithValue(b.MiningCtx, blockchain.CTX_PREV_HASH, prevHash)
			log.Infof("mining context: prev hash set to %s", prevHash.String())
		}
	}()
}
