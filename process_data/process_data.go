package processdata

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

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

// readFileProducer reads the file line by line and sends each line to the jobs channel for proessing by consumers
func readFileProducer(i int, ctx context.Context, filePath string, jobs chan<- []byte, errorCh chan *ValidateCouponCodeError, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("producer %d start reading\n", i)

	// open file
	file, err := os.Open(filePath)
	if err != nil {
		errorCh <- &ValidateCouponCodeError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to open file: %v", err)}
	}
	// close file when done
	defer file.Close()

	// scane file line by line
	scanner := bufio.NewScanner(file)

OuterLoop:
	for scanner.Scan() {

		// read line as bytes
		lineBytes := scanner.Bytes()
		line := make([]byte, len(lineBytes))
		copy(line, lineBytes)

		select {
		case <-ctx.Done():
			// stop reading if context is done
			break OuterLoop
		default:
			// send coupon code to jobs channel for processing
			jobs <- line
		}
	}

	fmt.Printf("producer %d done reading\n", i)

	// error on scan failure
	if err := scanner.Err(); err != nil {
		errorCh <- &ValidateCouponCodeError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to scan file: %v", err)}
	}

}

// processDataConsumer processes data sent from the producer. Consumer sends matching codes to results channel
func processDataConsumer(
	i int, ctx context.Context,
	jobs <-chan []byte,
	result chan []byte,
	couponCode string,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	fmt.Printf("consumer %d start processing\n", i)

	for {
		select {

		// stop processing if context is finished
		case <-ctx.Done():
			return
		case codeBytes, ok := <-jobs:

			// stop processing if no more jobs i.e. channel is closed
			if !ok {
				fmt.Printf("no more to process, consumer %d exiting\n", i)
				return
			}

			// send to result if match found
			if string(codeBytes) == couponCode {
				fmt.Printf("consumer %d -> match found %v\n", i, string(codeBytes))
				result <- codeBytes
				return
			}
		}
	}
}

// ValidateCouponCodeError custom error used to send in the response
type ValidateCouponCodeError struct {
	Code    int
	Message string
}

// Error implements the custom error
func (e *ValidateCouponCodeError) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Message)
}

// bufferSize for jobs channel
const bufferSize = 1000

// numWorker is number of consumers
const numWorker = 5

// threshold is the minumum number of valid codes found in files to be accepted as a valid coupon code
const threshold = 2

// ValidateCouponCode the main goroutine that validates the coupon code
func ValidateCouponCode(couponCode string) error {

	// return immediately for invalid coupon codes
	if len(couponCode) < 8 || len(couponCode) > 10 {
		return &ValidateCouponCodeError{
			Code:    http.StatusUnprocessableEntity,
			Message: "invalid coupon code, must be between 8 and 10 characters long",
		}
	}

	pwd, err := os.Getwd()
	if err != nil {
		return &ValidateCouponCodeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("error getting current working directory: %v", err),
		}
	}

	root := filepath.Join(pwd, "data")
	filePaths, err := getFilePaths(root)
	if err != nil {
		return &ValidateCouponCodeError{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("error getting data file paths: %v", err),
		}
	}

	// resultCh is a channel to send valid coupon codes, note size is 2 that expects at least 2 valid codes
	// a buffered channel will allow consumer to send result without blocking
	resultCh := make(chan []byte, threshold)

	// errorCh is a chanel to send any errors encountered in producer/consumer
	// is a buffered channel to be non-blocking
	errorCh := make(chan *ValidateCouponCodeError, 1)

	// jobsCh is a channel to send from the producer when reading the file. The consumer will receive from this channel and process.
	jobsCh := make(chan []byte, bufferSize)

	// context to signal to producer and consumer to finish
	ctx, cancel := context.WithCancel(context.Background())

	// cancel to signal consumers and producers to stop processing
	defer cancel()

	// wgConsumer is a WaitGroup to track the consumer goroutines
	var wgConsumer sync.WaitGroup

	// wgProducer is a WaitGroup to track the producer goroutines
	var wgProducer sync.WaitGroup

	// process data with multiple consumers
	for w := 1; w <= numWorker; w++ {
		wgConsumer.Add(1)
		go processDataConsumer(w, ctx, jobsCh, resultCh, couponCode, &wgConsumer)
	}

	// 1 producer per file to read line by line and send coupon code to jobs channel for consumer to process
	for i, file := range filePaths {
		wgProducer.Add(1)
		go readFileProducer(i, ctx, file, jobsCh, errorCh, &wgProducer)
	}

	// waiter goroutine: waits for all consumers to finish and close the results channel
	go func() {
		wgConsumer.Wait()

		// close results channel once finished
		close(resultCh)
		fmt.Println("done validating")
	}()

	// waiter goroutine: wait for all producers to finish reading files and closes the jobs and error channel
	// This will hande the case where no match is found and produces stop
	go func() {
		wgProducer.Wait()

		// close channel when done reading
		close(jobsCh)
		fmt.Println("done reading")

		// close error channel
		close(errorCh)
	}()

	// wait for match in result channel, finish if no match
	found := make(map[string]int)
	for range threshold {
		select {
		case resultBytes, ok := <-resultCh:
			if ok {
				result := string(resultBytes)
				found[result] += 1

				// if at least 2 matches are found, cancel the context immediately. This will signal produces and consumers to stop.
				if found[result] == 2 {
					fmt.Printf("valid %v\n", result)
					cancel()

					// return true for a valid coupon code
					return nil
				}
			}
		// if error is encountered, cancel the context which stops all producer/consumer
		case err := <-errorCh:
			if err != nil {
				cancel()
				return err
			}
		}
	}

	// return false for no matching coupon code
	fmt.Println("no match")
	return &ValidateCouponCodeError{
		Code:    http.StatusUnprocessableEntity,
		Message: "invalid coupon code, not found",
	}
}
