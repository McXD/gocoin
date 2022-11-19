package rpc

import (
	"github.com/gin-gonic/gin"
	"gocoin/blockchain"
	"gocoin/rpc/controllers"
)

func NewRouter(bc *blockchain.Blockchain) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// test
	ping := new(controllers.PingController)
	router.GET("/ping", ping.Test)

	// bcController
	bcController := controllers.BlockchainController{
		Blockchain: bc,
	}

	// wallet
	wallet := controllers.WalletController{
		DiskWallet: bc.DiskWallet,
	}

	router.GET("/blockchain/transactions", bcController.GetTransaction)
	router.GET("/wallet/info", wallet.GetWalletInfo)
	router.GET("/wallet/newAddress", wallet.GetNewAddress)
	router.GET("/wallet/listAddress", wallet.ListAddresses)
	router.GET("/wallet/listUnspent", wallet.ListUnspent)
	router.POST("/wallet/sendFrom", bcController.SendFrom)

	return router
}
