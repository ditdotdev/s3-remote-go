module github.com/datadatdat/s3-remote-go

require (
	github.com/aws/aws-sdk-go v1.30.5
	github.com/datadatdat/remote-sdk-go v0.2.4
	github.com/stretchr/testify v1.11.1
)

go 1.13

replace github.com/datadatdat/remote-sdk-go v0.2.4 => ../remote-sdk-go
