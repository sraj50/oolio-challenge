package main_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"oolio/backend-challenge/server"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetProduct(t *testing.T) {

	t.Run("GetProduct", func(t *testing.T) {
		router := gin.Default()
		router.GET("/product", server.GetProductHandler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/product", nil)
		router.ServeHTTP(w, req)

		want := server.Products

		var got []server.Product
		err := json.NewDecoder(w.Body).Decode(&got)
		if err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, want, got)
	})
}

func TestGetProductId(t *testing.T) {

	t.Run("GetProductId", func(t *testing.T) {
		router := gin.Default()
		router.GET("/product/:id", server.GetProductIdHandler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/product/1", nil)
		router.ServeHTTP(w, req)

		want := server.Product{
			ID:       "1",
			Name:     "Chicken Waffle",
			Category: "Waffle",
			Price:    6.5,
		}

		var got server.Product
		err := json.NewDecoder(w.Body).Decode(&got)
		if err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, want, got)
	})
}

func TestPostOrder(t *testing.T) {
	testCases := []struct {
		req     server.PostOrderRequest
		resCode int
	}{
		{req: server.PostOrderRequest{}, resCode: http.StatusBadRequest},                                                                                          // empty body, 400 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{}}, resCode: http.StatusBadRequest},                                                               // empty items, 400 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "", Quantity: 1}}}, resCode: http.StatusBadRequest},                                   // empty productId, 400 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1"}}}, resCode: http.StatusBadRequest},                                               // no quantity, 400 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1", Quantity: 0}}}, resCode: http.StatusBadRequest},                                  // 0 quantity, 400 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1", Quantity: 1}}}, resCode: http.StatusOK},                                          // no coupon code, 200 ok
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1", Quantity: 1}}, CouponCode: "blah"}, resCode: http.StatusUnprocessableEntity},     // invalid coupon code, does not meet character requirements, 422 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1", Quantity: 1}}, CouponCode: "SUPER100"}, resCode: http.StatusUnprocessableEntity}, // coupon code, fails validation requirements, 422 bad request
		{req: server.PostOrderRequest{Items: []server.OrderItem{{ProductId: "1", Quantity: 1}}, CouponCode: "FIFTYOFF"}, resCode: http.StatusOK},                  // coupon code, fails validation requirements, 200 ok
	}

	t.Run("PostOrder", func(t *testing.T) {
		router := gin.Default()
		router.POST("/order", server.PostOrderHandler)

		for _, tc := range testCases {

			reqBodyJson, err := json.Marshal(tc.req)
			if err != nil {
				t.Fatalf("failed to marshal post request body: %v", err)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/order", strings.NewReader(string(reqBodyJson)))
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.resCode, w.Code)

		}
	})
}
