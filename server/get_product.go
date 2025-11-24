package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// product represents data about a product
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float32 `json:"price"`
}

var Products = []Product{
	{ID: "1", Name: "Chicken Waffle", Category: "Waffle", Price: 6.5},
	{ID: "2", Name: "Vanilla Bean Crème Brûlée", Category: "Crème Brûlée", Price: 7},
	{ID: "3", Name: "Macaron Mix of Five", Category: "Macaron", Price: 8},
	{ID: "4", Name: "Classic Tiramisu", Category: "Tiramisu", Price: 5.5},
	{ID: "5", Name: "Pistachio Baklava", Category: "Baklava", Price: 4},
}

// GetProductHandler returns products as JSON
func GetProductHandler(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, Products)
}
