package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// GetProductIdHandler returns a specific product by id as JSON
func GetProductIdHandler(c *gin.Context) {
	id := c.Param("id")

	// loop over list products to find the match by id
	for _, p := range Products {
		if p.ID == id {
			c.IndentedJSON(http.StatusOK, p)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "product not found"})
}
