package server

import (
	"errors"
	"net/http"
	"oolio/backend-challenge/process_data"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderItem struct {
	ProductId string `json:"productId" binding:"required,min=1,ne= "`
	Quantity  int32  `json:"quantity" binding:"required,gt=0"`
}

type PostOrderRequest struct {
	Items      []OrderItem `json:"items" binding:"required,min=1,dive"`
	CouponCode string      `json:"couponCode"`
}

type PostOrderResponse struct {
	ID       string      `json:"id"`
	Items    []OrderItem `json:"items"`
	Products []Product   `json:"products"`
}

// PostOrderHandler to add a new
func PostOrderHandler(c *gin.Context) {
	var body PostOrderRequest

	// bind the request body to PostOrderRequest, return error for invalid body
	if err := c.ShouldBindJSON(&body); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate the coupon code
	validationError := processdata.ValidateCouponCode(body.CouponCode)
	var verr *processdata.ValidateCouponCodeError

	// return error for invalid coupon code
	if errors.As(validationError, &verr) {
		c.IndentedJSON(verr.Code, gin.H{"message": verr.Message})
		return
	}

	productsToOrder := []Product{}
	// loop over list products to find the match by id
	for _, p := range Products {
		for _, o := range body.Items {
			if o.ProductId == p.ID {
				productsToOrder = append(productsToOrder, p)
				break
			}
		}
	}

	response := PostOrderResponse{
		ID:       uuid.NewString(),
		Items:    body.Items,
		Products: productsToOrder,
	}

	c.IndentedJSON(http.StatusOK, response)
}
