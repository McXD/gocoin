package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type PingController struct{}

func (t *PingController) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
