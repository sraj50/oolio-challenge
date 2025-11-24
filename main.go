package main

import (
	"github.com/gin-gonic/gin"
	"oolio/backend-challenge/server"
)

func main() {
	router := gin.Default()

	router.GET("/product", server.GetProductHandler)
	router.GET("/product/:id", server.GetProductIdHandler)
	router.POST("/order", server.PostOrderHandler)

	router.Run("localhost:8080")
}
