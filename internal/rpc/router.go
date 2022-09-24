package rpc

import (
	"github.com/gin-gonic/gin"
	"gocoin/internal/rpc/controllers"
)

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	ping := new(controllers.PingController)

	router.GET("/ping", ping.Test)

	return router
}
