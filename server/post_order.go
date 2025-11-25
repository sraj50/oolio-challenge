package server

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
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

// getFilePaths gets all file paths in the specified directory
func getFilePaths(rootDir string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			paths = append(paths, absPath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return paths, nil
}

// readFileProducer reads the file line by line and sends each line to channel
func readFileProducer(i int, ctx context.Context, filePath string, jobs chan<- string, wg *sync.WaitGroup) {
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

	fmt.Printf("producer %d done reading\n", i)

	// error on scan failure
	if err := scanner.Err(); err != nil {
		log.Fatalf("failed to scan file: %v", err)
	}

}

// processDataConsumer processes data sent from producer
func processDataConsumer(i int, ctx context.Context, jobs <-chan string, result chan string, couponCode string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {

		// stop processing if context is finished
		case <-ctx.Done():
			return
		case code, ok := <-jobs:

			// stop processing if no more jobs
			if !ok {
				fmt.Printf("no more to process, consumer %d exiting\n", i)
				return
			}

			// send to result if match found
			if code == couponCode {
				fmt.Printf("worker %d -> match found %v\n", i, code)
				result <- code
				return
			}
		}
	}
}

func validateCouponCode(couponCode string) bool {

	if len(couponCode) < 8 || len(couponCode) > 10 {
		return false
	}

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("error getting current working directory: %v", err)
	}

	root := filepath.Join(pwd, "data")
	filePaths, err := getFilePaths(root)
	if err != nil {
		log.Fatalf("error getting data file paths: %v", err)
	}
	for _, f := range filePaths {
		fmt.Println(f)
	}

	// relativePath := "data/couponbase1"

	// file := filepath.Join(pwd, relativePath)

	const bufferSize = 1000
	const numWorker = 5

	// channel to store valid coupon code, note size is 1 imples is a buffered channel
	// this will allow consumer to send result without blocking
	resultCh := make(chan string, 2)

	// channel to send from producer when reading file. Consumer will read from this channel and process.
	dataCh := make(chan string, bufferSize)

	// context to signal to producer and consumer to finish
	ctx, cancel := context.WithCancel(context.Background())

	// cancel to signal consumers and producers to stop processing
	defer cancel()

	// track go routines
	var wgConsumer sync.WaitGroup
	var wgProducer sync.WaitGroup

	// process data with multiple consumers
	for w := 1; w <= numWorker; w++ {
		wgConsumer.Add(1)
		go processDataConsumer(w, ctx, dataCh, resultCh, couponCode, &wgConsumer)
	}

	// 1 producer per file to read line by line and send coupon code to channel for consumer to process
	for i, file := range filePaths {
		wgProducer.Add(1)
		go readFileProducer(i, ctx, file, dataCh, &wgProducer)
	}

	// waiter goroutine: waits for all consumers and producers to finish, then close result
	go func() {
		wgConsumer.Wait()

		// close results channel once finished
		close(resultCh)
		fmt.Println("done validating")

		// cancel
		// cancel()
		// fmt.Println("finished!")
	}()

	go func() {
		wgProducer.Wait()

		// close channel when done reading
		close(dataCh)
		fmt.Println("done reading")
	}()

	// wait for match in result channel, finish if no match
	found := make(map[string]int)
	for result := range resultCh {
		found[result] += 1
		if found[result] == 2 {
			fmt.Printf("valid %v\n", result)
			cancel()
			return true
		}
	}

	fmt.Println("no match")

	// return the matching coupon code
	return false
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
