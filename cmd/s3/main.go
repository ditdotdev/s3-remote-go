// Package main provides the S3 remote server for Titan data storage.
// Package main provides the S3 remote server for Titan data storage.
package main

import "github.com/datadatdat/remote-sdk-go/remote"

func main() {
	remote.Serve("s3")
}
