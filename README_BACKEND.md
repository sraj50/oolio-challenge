# Local development

In the project root, create a directory named `data`.
Copy the following files:
- [file1](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase1.gz)
- [file2](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase2.gz)
- [file3](https://orderfoodonline-files.s3.ap-southeast-2.amazonaws.com/couponbase3.gz)

Run `go get .` to install project dependencies.

Run `go run .` to run the server listening on `localhost:8080`

Run `go test ./...` to run tests.