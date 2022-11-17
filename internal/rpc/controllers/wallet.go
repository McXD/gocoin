package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type WalletController struct{}

func (t *WalletController) GetWalletInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (t *WalletController) GetNewAddress(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (t *WalletController) ListUnspent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (t *WalletController) SendTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
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
