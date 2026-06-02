// Package main provides the S3 remote server for Dit data storage.
// Package main provides the S3 remote server for Dit data storage.
package main

import "github.com/ditdotdev/remote-sdk-go/remote"

func main() {
	remote.Serve("s3")
}
