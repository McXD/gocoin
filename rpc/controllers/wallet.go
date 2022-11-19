package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gocoin/core"
	"gocoin/wallet"
	"net/http"
)

type WalletController struct {
	*wallet.DiskWallet
}

// map of address to balance
type walletInfo map[string]uint32

// GetWalletInfo returns overview of the wallet
// GET /wallet/info
func (t *WalletController) GetWalletInfo(c *gin.Context) {
	balances := t.DiskWallet.GetBalances()
	info := make(walletInfo, len(balances))
	for addr, balance := range balances {
		info[addr.String()] = balance
	}

	c.JSON(http.StatusOK, info)
}

func (t *WalletController) ListAddresses(c *gin.Context) {
	addresses := t.DiskWallet.ListAddresses()
	rets := make([]string, len(addresses))
	for i, addr := range addresses {
		rets[i] = addr.String()
	}

	c.JSON(http.StatusOK, rets)
}

// GetNewAddress generates and returns a new address
// GET /wallet/newAddress
func (t *WalletController) GetNewAddress(c *gin.Context) {
	address, err := t.NewAddress()
	if err != nil {
		SendError(c, http.StatusInternalServerError, fmt.Errorf("failed to generate new address: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address.String(),
	})
}

type uxtoDTO struct {
	TxId    string `json:"txid"`
	Vout    uint32 `json:"vout"`
	Address string `json:"address"`
	Amount  uint32 `json:"amount"`
}

// ListUnspent lists all unspent outputs for the given address
// GET /wallet/listUnspent?address=1JwSSubhmg6iPtRjtyqhUYYH7bZg3Lfy1T
func (t *WalletController) ListUnspent(c *gin.Context) {
	address := core.Hash160{}

	if err := address.ParseAddress(c.Query("address")); err != nil {
		SendError(c, http.StatusBadRequest, fmt.Errorf("failed to parse address: %w", err))
		return
	}

	unspent, err := t.DiskWallet.ListUnspent(address)
	if err != nil {
		SendError(c, http.StatusInternalServerError, fmt.Errorf("failed to get unspent UXTOs from wallet: %w", err))
	}

	rets := make([]uxtoDTO, len(unspent))
	for i, u := range unspent {
		rets[i] = uxtoDTO{
			TxId:    u.TxId.String(),
			Vout:    u.N,
			Address: u.PubKeyHash.String(),
			Amount:  u.Value,
		}
	}

	c.JSON(http.StatusOK, rets)
}

func (t *WalletController) CreateRawTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (t *WalletController) SignRawTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (t *WalletController) SendRawTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
