/*
 * Copyright The Titan Project Contributors.
 */
package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/mock"
)

type MockProvider struct {
	mock.Mock
}

func (p *MockProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	args := p.Called(ctx)
	return args.Get(0).(aws.Credentials), args.Error(1)
}

type MockS3 struct {
	err error
	*s3.GetObjectInput
	s3.GetObjectOutput
}

func (m *MockS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	m.GetObjectInput = in
	return &m.GetObjectOutput, m.err
}
