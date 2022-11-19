package controllers

import "github.com/gin-gonic/gin"

func JSONError(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}

func SendError(c *gin.Context, code int, err error) {
	c.JSON(code, JSONError(err))
}
