# Local development

In the project root, create a directory named `data`.
Copy the following files:
- [file1](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase1.gz)
- [file2](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase2.gz)
- [file3](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase3.gz)

Run `go get .` to install project dependencies.

Run `go run .` to run the server listening on `localhost:8080`

Run `go test ./...` to run tests.

# Design Overview

A producer–consumer concurrency pattern.

A producer goroutine reads each file line by line and sends each coupon code to a shared “jobs” channel.

Multiple consumer goroutines read the jobs channel:

- Each consumer compares the received code with the user-provided coupon code.

- If the code matches, it sends the code to a results channel.

## Execution and Synchronization

A WaitGroup is maintained to track all producer and consumer goroutines.

The main goroutine listens to the results channel and tracks matches in a map[string]int. The map counts how many times (or in how many files) the code has been found.

When the count for a code reaches 2 (i.e. the coupon appears in at least two files), we immediately stop all ongoing work:

- Cancel via context.Cancel — signalling producers and consumers to stop.

- Close channels and return success.

## Handling “No-Match” Scenario

If no matching code is found:

- Producers will keep reading until EOF for each file, then finish.

- Consumers will exit when the jobs channel is closed and no more data is available.

- Once all goroutines finish, the results channel is closed. The main goroutine detects that there are no further results and returns “no match.”

## Error Handling

In addition to the result channel, there is an error channel for producers or consumers to report any fatal errors (e.g. file open failure, scan error).

Upon receiving an error, the main goroutine cancels all work immediately and propagates the error (or handles it appropriately).

## Testing

Tests are included to validate various scenarios — for example:

- Code found in two (or more) files → success.

- Code appears once (or not at all) → returns “no match.”

- Invalid POST body requests.