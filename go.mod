module github.com/datadatdat/s3-remote-go

require (
	github.com/aws/aws-sdk-go v1.55.8
	github.com/datadatdat/remote-sdk-go v0.2.4
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2 // indirect
)

go 1.13

replace github.com/datadatdat/remote-sdk-go v0.2.4 => ../remote-sdk-go
