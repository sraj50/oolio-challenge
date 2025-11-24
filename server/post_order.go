package server

import (
	"bufio"
	"context"
	"fmt"

	// "fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

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

// readFileProducer reads the file line by line and sends each line to channgel
func readFileProducer(ctx context.Context, filePath string, jobs chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	// open file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	// close file when done
	defer file.Close()

	// scane file line by line
	scanner := bufio.NewScanner(file)

OuterLoop:
	for scanner.Scan() {

		code := scanner.Text()

		select {
		case <-ctx.Done():
			// stop reading if coupon code found
			break OuterLoop
		default:
			// send couponConde to jobs channel for processing
			jobs <- code
		}
	}

	// error on scan failure
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to scan file: %v", err)
	}

	// close channel when done reading
	close(jobs)
	fmt.Println("done reading")
}

// processDataConsumer processes data sent from producer
func processDataConsumer(ctx context.Context, jobs <-chan string, result chan bool, couponCode string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {

		// stop processing if context is finished
		case <-ctx.Done():
			return
		case code, ok := <-jobs:

			// stop processing if no more jobs
			if !ok {
				return
			}

			// send to result if match found
			if code == couponCode {
				fmt.Printf("match found %v\n", code)
				result <- true
				return
			}
		}
	}
}

func validateCouponCode(couponCode string) bool {
	relativePath := "data/couponbase1"
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current working directory: %v", err)
	}
	filePath := filepath.Join(pwd, relativePath)

	const bufferSize = 1000
	const numWorker = 5

	// channel to store valid coupon code, note size is 1 imples is a buffered channel
	// this will allow consumer to send result without blocking
	resultCh := make(chan bool, 1)

	// channel to send from producer when reading file. Consumer will read from this channel and process.
	dataCh := make(chan string, bufferSize)

	// context to signal to producer and consumer to finish
	ctx, cancel := context.WithCancel(context.Background())

	// cancel to signal consumers and producers to stop processing
	defer cancel()

	// track go routines
	var wg sync.WaitGroup

	// process data with multiple consumers
	for w := 1; w <= numWorker; w++ {
		wg.Add(1)
		go processDataConsumer(ctx, dataCh, resultCh, couponCode, &wg)
	}

	// producer reading line by line and sending coupon code to channel for consumer to process
	wg.Add(1)
	go readFileProducer(ctx, filePath, dataCh, &wg)

	// waiter goroutine: waits for all consumers and producers to finish, then close result
	go func() {
		wg.Wait()

		// close results channel once finished
		close(resultCh)
	}()

	// wait for match in result channel, finish if no match
	found := false
	result, ok := <-resultCh
	if ok {
		found = result
	} else {
		// closed channel implies no match was found
		fmt.Println("no match found")
	}

	// return the matching coupon code
	return found
}

// PostOrderHandler to add a new
func PostOrderHandler(c *gin.Context) {
	var body PostOrderRequest

	if err := c.ShouldBindJSON(&body); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isValidCouponCode := validateCouponCode(body.CouponCode)

	if !isValidCouponCode {
		c.IndentedJSON(http.StatusUnprocessableEntity, gin.H{"message": "invalid coupon code"})
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
