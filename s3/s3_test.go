/*
 * Copyright Dit.
 */
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/ditdotdev/remote-sdk-go/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testAccessUpper = "ACCESS"
	testSecretUpper = "SECRET"
	testRegionUpper = "REGION"
	testAccessLower = "access"
	testSecretLower = "secret"
	testBar         = "bar"
	testFoo         = "foo"
	testToken       = "token"
)

func TestRegistered(t *testing.T) {
	r, _ := remote.Get("s3")
	ret, err := r.Type()

	if assert.NoError(t, err) {
		assert.Equal(t, "s3", ret)
	}
}

func TestFromURL(t *testing.T) {
	r, _ := remote.Get("s3")
	props, err := r.FromURL("s3://bucket/object/path", map[string]string{})

	if assert.NoError(t, err) {
		assert.Equal(t, propBucket, props[propBucket])
		assert.Equal(t, "object/path", props[propPath])
		assert.Nil(t, props[propAccessKey])
		assert.Nil(t, props[propSecretKey])
		assert.Nil(t, props[propRegion])
	}
}

func TestNoPath(t *testing.T) {
	r, _ := remote.Get("s3")
	props, err := r.FromURL("s3://bucket", map[string]string{})

	if assert.NoError(t, err) {
		assert.Equal(t, propBucket, props[propBucket])
		assert.Nil(t, props[propPath])
		assert.Nil(t, props[propAccessKey])
		assert.Nil(t, props[propSecretKey])
		assert.Nil(t, props[propRegion])
	}
}

func TestBadUrl(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://host\nname", map[string]string{})
	assert.Error(t, err)
}

func TestBadScheme(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3", map[string]string{})
	assert.Error(t, err)
}

func TestBadSchemeName(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("foo://bucket/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadProperty(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{testFoo: testBar})
	assert.Error(t, err)
}

func TestBadUser(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://user@bucket/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadUserPassword(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://user:password@bucket/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadPort(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://bucket:80/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadMissingBucket(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3:///object/path", map[string]string{})
	assert.Error(t, err)
}

func TestProperties(t *testing.T) {
	r, _ := remote.Get("s3")
	props, err := r.FromURL("s3://bucket/object/path", map[string]string{
		propAccessKey: testAccessUpper, propSecretKey: testSecretUpper, propRegion: testRegionUpper,
	})

	if assert.NoError(t, err) {
		assert.Equal(t, propBucket, props[propBucket])
		assert.Equal(t, "object/path", props[propPath])
		assert.Equal(t, testAccessUpper, props[propAccessKey])
		assert.Equal(t, testSecretUpper, props[propSecretKey])
		assert.Equal(t, testRegionUpper, props[propRegion])
	}
}

func TestBadAccessKeyOnly(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{propAccessKey: testAccessUpper})
	assert.Error(t, err)
}

func TestBadSecretKeyOnly(t *testing.T) {
	r, _ := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{propSecretKey: testAccessUpper})
	assert.Error(t, err)
}

func TestToURL(t *testing.T) {
	r, _ := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{propBucket: propBucket, propPath: propPath})

	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Empty(t, props)
	}
}

func TestWithKeys(t *testing.T) {
	r, _ := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{
		propBucket: propBucket, propPath: propPath,
		propAccessKey: testAccessUpper, propSecretKey: testSecretUpper,
	})

	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Len(t, props, 2)
		assert.Equal(t, testAccessUpper, props[propAccessKey])
		assert.Equal(t, testSecretUpper, props[propSecretKey])
	}
}

func TestWithRegion(t *testing.T) {
	r, _ := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{
		propBucket: propBucket, propPath: propPath,
		propRegion: testRegionUpper,
	})

	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Len(t, props, 1)
		assert.Equal(t, testRegionUpper, props[propRegion])
	}
}

func TestGetParameters(t *testing.T) {
	r, _ := remote.Get("s3")
	props, err := r.GetParameters(map[string]interface{}{
		propBucket: propBucket, propPath: propPath,
		propAccessKey: testAccessUpper, propSecretKey: testSecretUpper, propRegion: testRegionUpper,
	})

	if assert.NoError(t, err) {
		assert.Equal(t, testAccessUpper, props[propAccessKey])
		assert.Equal(t, testSecretUpper, props[propSecretKey])
		assert.Equal(t, testRegionUpper, props[propRegion])
	}
}

func TestGetParametersEnvironment(t *testing.T) {
	r, _ := remote.Get("s3")

	_ = os.Setenv("AWS_ACCESS_KEY_ID", testAccessUpper)

	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", testSecretUpper)

	_ = os.Setenv("AWS_REGION", "us-west-2")

	_ = os.Setenv("AWS_SESSION_TOKEN", "TOKEN")
	props, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})

	if assert.NoError(t, err) {
		assert.Equal(t, testAccessUpper, props[propAccessKey])
		assert.Equal(t, testSecretUpper, props[propSecretKey])
		assert.Equal(t, "us-west-2", props[propRegion])
		assert.Equal(t, "TOKEN", props[propSessionToken])
	}
}

func TestGetParametersFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "s3.test")

	if !assert.NoError(t, err) {
		return
	}

	defer func() { _ = os.RemoveAll(dir) }()

	_ = os.Unsetenv("AWS_ACCESS_KEY_ID")

	_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	_ = os.Unsetenv("AWS_REGION")

	_ = os.Unsetenv("AWS_SESSION_TOKEN")

	configFile := fmt.Sprintf("%s/config", dir)

	credFile := fmt.Sprintf("%s/credentials", dir)

	configContent := `
[default]
region = us-west-1
`

	//nolint:gosec // Test credentials only
	credContent := `
[default]
aws_access_key_id = ACCESS2
aws_secret_access_key = SECRET2
aws_session_token = TOKEN2
`

	err1 := os.WriteFile(configFile, []byte(configContent), 0o600)

	err2 := os.WriteFile(credFile, []byte(credContent), 0o600)

	if assert.NoError(t, err1) && assert.NoError(t, err2) {
		_ = os.Setenv("AWS_CONFIG_FILE", configFile)
		_ = os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)

		r, _ := remote.Get("s3")

		props, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})
		if assert.NoError(t, err) {
			assert.Equal(t, "ACCESS2", props[propAccessKey])
			assert.Equal(t, "SECRET2", props[propSecretKey])
			assert.Equal(t, "us-west-1", props[propRegion])
			assert.Equal(t, "TOKEN2", props[propSessionToken])
		}
	}
}

func TestBadNewSession(t *testing.T) {
	r, _ := remote.Get("s3")

	// For AWS SDK v2, we need to mock the config loading

	originalNewConfig := newConfig

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("config load error")
	}

	defer func() {
		newConfig = originalNewConfig
	}()

	_, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})
	assert.Error(t, err)
}

func TestBadConfigCredentials(t *testing.T) {
	r, _ := remote.Get("s3")

	p := new(MockProvider)
	p.On("Retrieve", mock.Anything).Return(aws.Credentials{}, errors.New("err"))

	originalNewConfig := newConfig

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{
			Credentials: aws.CredentialsProviderFunc(p.Retrieve),
		}, nil
	}

	defer func() {
		newConfig = originalNewConfig
	}()

	_, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})
	assert.Error(t, err)
}

func TestBadCredentialsAccessKey(t *testing.T) {
	r, _ := remote.Get("s3")

	p := new(MockProvider)
	p.On("Retrieve", mock.Anything).Return(aws.Credentials{}, nil)

	originalNewConfig := newConfig

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{
			Credentials: aws.CredentialsProviderFunc(p.Retrieve),
		}, nil
	}

	defer func() {
		newConfig = originalNewConfig
	}()

	_, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})
	assert.Error(t, err)
}

func TestBadCredentialsRegion(t *testing.T) {
	r, _ := remote.Get("s3")

	p := new(MockProvider)
	p.On("Retrieve", mock.Anything).Return(aws.Credentials{
		AccessKeyID:     testAccessUpper,
		SecretAccessKey: testSecretUpper,
	}, nil)

	originalNewConfig := newConfig

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{
			Credentials: aws.CredentialsProviderFunc(p.Retrieve),
		}, nil
	}

	defer func() {
		newConfig = originalNewConfig
	}()

	_, err := r.GetParameters(map[string]interface{}{propBucket: propBucket, propPath: propPath})
	assert.Error(t, err)
}

func TestMetadataKey(t *testing.T) {
	k := getMetadataKey(aws.String(testFoo))
	assert.Equal(t, "foo/dit", k)
}

func TestMetadataKeyNil(t *testing.T) {
	k := getMetadataKey(nil)
	assert.Equal(t, "dit", k)
}

func TestKeyNoPath(t *testing.T) {
	k := getKey(map[string]interface{}{}, aws.String("id"))
	assert.Equal(t, "id", *k)
}

func TestKeyNoPathNoCommit(t *testing.T) {
	k := getKey(map[string]interface{}{}, nil)
	assert.Nil(t, k)
}

func TestKeyPath(t *testing.T) {
	k := getKey(map[string]interface{}{propPath: "one/two"}, aws.String("three"))
	assert.Equal(t, "one/two/three", *k)
}

func TestKeyPathNoCommit(t *testing.T) {
	k := getKey(map[string]interface{}{propPath: "one/two"}, nil)
	assert.Equal(t, "one/two", *k)
}

func TestKeyPathNotString(t *testing.T) {
	k := getKey(map[string]interface{}{propPath: 123}, aws.String("id"))
	assert.Equal(t, "id", *k)
}

func TestGetMetadataBadBucket(t *testing.T) {
	setMockS3(t, &MockS3{})
	_, err := getMetadataContent(map[string]interface{}{propBucket: 123}, map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket must be a string")
}

func TestGetCommitBadBucket(t *testing.T) {
	setMockS3(t, &MockS3{})
	r, _ := remote.Get("s3")
	_, err := r.GetCommit(map[string]interface{}{propBucket: 123}, map[string]interface{}{}, "id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bucket must be a string")
}

func TestValidateRemoteAllProperties(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{
		propBucket: propBucket, propSecretKey: testSecretLower,
		propAccessKey: testAccessLower, propPath: "/path", propRegion: propRegion,
	})
	assert.NoError(t, err)
}

func TestValidateRemoteOnlyRequired(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{propBucket: propBucket})
	assert.NoError(t, err)
}

func TestValidateRemoteMissingRequired(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{})
	assert.Error(t, err)
}

func TestValidateRemoteInvalidPoperty(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{propBucket: propBucket, testFoo: testBar})
	assert.Error(t, err)
}

func TestValidateRemoteOnlyAccessKey(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{propBucket: propBucket, propAccessKey: testAccessLower})
	assert.Error(t, err)
}

func TestValidateRemoteOnlySecretKey(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateRemote(map[string]interface{}{propBucket: propBucket, propSecretKey: testSecretLower})
	assert.Error(t, err)
}

func TestValidateParametersEmpty(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateParameters(map[string]interface{}{})
	assert.NoError(t, err)
}

func TestValidateParametersAll(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateParameters(map[string]interface{}{
		propAccessKey: testAccessLower, propSecretKey: testSecretLower,
		propRegion: propRegion, propSessionToken: testToken,
	})
	assert.NoError(t, err)
}

func TestValidateParametersInvalid(t *testing.T) {
	r, _ := remote.Get("s3")

	err := r.ValidateParameters(map[string]interface{}{testFoo: testBar})
	assert.Error(t, err)
}

var mockConfig aws.Config

// setMockS3 installs a mock S3 client and registers cleanup so the global is
// always reset, even when an assertion fails mid-test. This prevents state from
// leaking into subsequent tests.
func setMockS3(t *testing.T, m ClientInterface) {
	t.Helper()

	mockS3 = m

	t.Cleanup(func() { mockS3 = nil })
}

// installMockS3 installs mock implementations of newConfig and newS3Client and
// registers cleanup so the globals are always restored.
func installMockS3(t *testing.T) {
	t.Helper()

	origConfig := newConfig
	origClient := newS3Client

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return mockConfig, nil
	}

	newS3Client = func(_ aws.Config, _ ...func(*s3.Options)) *s3.Client {
		return &s3.Client{}
	}

	t.Cleanup(func() {
		newConfig = origConfig
		newS3Client = origClient
	})
}

func TestGetS3(t *testing.T) {
	installMockS3(t)

	mockConfig = aws.Config{
		Region: propRegion,
		Credentials: aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     testAccessLower,
				SecretAccessKey: testSecretLower,
			}, nil
		}),
	}

	_, err := getS3(map[string]interface{}{propAccessKey: testAccessLower, propSecretKey: testSecretLower, propRegion: propRegion},
		map[string]interface{}{})

	if assert.NoError(t, err) {
		assert.Equal(t, propRegion, mockConfig.Region)

		creds, err := mockConfig.Credentials.Retrieve(context.Background())
		if assert.NoError(t, err) {
			assert.Equal(t, testAccessLower, creds.AccessKeyID)
			assert.Equal(t, testSecretLower, creds.SecretAccessKey)
		}
	}
}

func TestGetS3Parameters(t *testing.T) {
	installMockS3(t)

	mockConfig = aws.Config{
		Region: propRegion,
		Credentials: aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     testAccessLower,
				SecretAccessKey: testSecretLower,
				SessionToken:    testToken,
			}, nil
		}),
	}

	_, err := getS3(map[string]interface{}{propBucket: propBucket},
		map[string]interface{}{propAccessKey: testAccessLower, propSecretKey: testSecretLower, propRegion: propRegion, propSessionToken: testToken})

	if assert.NoError(t, err) {
		assert.Equal(t, propRegion, mockConfig.Region)

		creds, err := mockConfig.Credentials.Retrieve(context.Background())
		if assert.NoError(t, err) {
			assert.Equal(t, testAccessLower, creds.AccessKeyID)
			assert.Equal(t, testSecretLower, creds.SecretAccessKey)
			assert.Equal(t, testToken, creds.SessionToken)
		}
	}
}

func TestGetS3MissingRegion(t *testing.T) {
	installMockS3(t)

	_, err := getS3(map[string]interface{}{propBucket: propBucket},
		map[string]interface{}{propAccessKey: testAccessLower, propSecretKey: testSecretLower})
	assert.Error(t, err)
}

func TestGetS3MissingAccessKey(t *testing.T) {
	installMockS3(t)

	_, err := getS3(map[string]interface{}{propBucket: propBucket},
		map[string]interface{}{propRegion: propRegion, propSecretKey: testSecretLower})
	assert.Error(t, err)
}

func TestGetS3MissingSecretKey(t *testing.T) {
	installMockS3(t)

	_, err := getS3(map[string]interface{}{propBucket: propBucket},
		map[string]interface{}{propRegion: propRegion, propAccessKey: testAccessLower})
	assert.Error(t, err)
}

func TestGetS3BadToken(t *testing.T) {
	installMockS3(t)

	_, err := getS3(map[string]interface{}{propBucket: propBucket},
		map[string]interface{}{propAccessKey: testAccessLower, propSecretKey: testSecretLower, propRegion: propRegion, propSessionToken: 4})
	assert.Error(t, err)
}

func TestGetS3BadRemote(t *testing.T) {
	installMockS3(t)

	_, err := getS3(map[string]interface{}{propBucket: propBucket, propAccessKey: 4},
		map[string]interface{}{propSecretKey: testSecretLower, propRegion: propRegion})
	assert.Error(t, err)
}

func TestNewConfigFails(t *testing.T) {
	originalNewConfig := newConfig

	newConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("config error")
	}

	t.Cleanup(func() { newConfig = originalNewConfig })

	_, err := getS3(map[string]interface{}{propAccessKey: testAccessLower, propSecretKey: testSecretLower, propRegion: propRegion},
		map[string]interface{}{})
	assert.Error(t, err)
}

func TestInstallMock(t *testing.T) {
	m := &MockS3{}
	setMockS3(t, m)
	svc, _ := getS3(map[string]interface{}{}, map[string]interface{}{})
	assert.Equal(t, ClientInterface(m), svc)
}

func TestGetMetadataContent(t *testing.T) {
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader("metadata")),
		},
	})

	res, err := getMetadataContent(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{})

	if assert.NoError(t, err) {
		content, err := io.ReadAll(res)
		if assert.NoError(t, err) {
			assert.Equal(t, "metadata", string(content))
		}
	}
}

func TestGetMetadataGetS3Error(t *testing.T) {
	installMockS3(t)

	_, err := getMetadataContent(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{})
	assert.Error(t, err)
}

func TestGetMetadataMissing(t *testing.T) {
	setMockS3(t, &MockS3{
		err: &types.NoSuchKey{},
	})
	res, err := getMetadataContent(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{})

	if assert.NoError(t, err) {
		content, err := io.ReadAll(res)
		if assert.NoError(t, err) {
			assert.Equal(t, "", string(content))
		}
	}
}

func TestGetMetadataOtherError(t *testing.T) {
	setMockS3(t, &MockS3{
		err: &types.NoSuchBucket{},
	})
	_, err := getMetadataContent(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{})
	assert.Error(t, err)
}

func TestListCommits(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z"}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`

	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(metadata)),
		},
	})

	r, _ := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, []remote.Tag{})

	if assert.NoError(t, err) {
		assert.Len(t, commits, 2)
		assert.Equal(t, "two", commits[0].ID)
		assert.Equal(t, "one", commits[1].ID)
	}
}

func TestListCommitsInvalid(t *testing.T) {
	metadata := `
foo
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`

	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(metadata)),
		},
	})

	r, _ := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, []remote.Tag{})

	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "two", commits[0].ID)
	}
}

func TestListCommitsTags(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z", "tags": { "a": "b" }}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}`

	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(metadata)),
		},
	})

	r, _ := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})

	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "one", commits[0].ID)
	}
}

func TestListCommitsError(t *testing.T) {
	setMockS3(t, &MockS3{
		err: errors.New("error"),
	})

	r, _ := remote.Get("s3")
	_, err := r.ListCommits(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})
	assert.Error(t, err)
}

// trackingReadCloser wraps a Reader and records whether Close was called.
type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

// errReader returns a non-EOF error after delivering some bytes, so that
// bufio.Scanner.Err() surfaces a non-nil error after the scan loop.
type errReader struct {
	data []byte
	pos  int
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, r.err
	}

	n := copy(p, r.data[r.pos:])
	r.pos += n

	return n, nil
}

func TestListCommitsClosesMetadataBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(
		`{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z"}}`,
	)}
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: body,
		},
	})

	r, _ := remote.Get("s3")
	_, err := r.ListCommits(
		map[string]interface{}{propBucket: propBucket, propPath: propPath},
		map[string]interface{}{}, []remote.Tag{},
	)

	if assert.NoError(t, err) {
		assert.True(t, body.closed, "metadata body must be closed after ListCommits")
	}
}

func TestListCommitsScannerError(t *testing.T) {
	// Reader returns a non-EOF error mid-read so scanner.Err() surfaces it.
	body := io.NopCloser(&errReader{
		data: []byte(`{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z"}}` + "\n"),
		err:  errors.New("simulated read failure"),
	})
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: body,
		},
	})

	r, _ := remote.Get("s3")
	_, err := r.ListCommits(
		map[string]interface{}{propBucket: propBucket, propPath: propPath},
		map[string]interface{}{}, []remote.Tag{},
	)
	assert.Error(t, err, "scanner.Err() should be surfaced after the scan loop")
	assert.Contains(t, err.Error(), "simulated read failure")
}

func TestGetCommitBadS3(t *testing.T) {
	installMockS3(t)

	r, _ := remote.Get("s3")
	_, err := r.GetCommit(map[string]interface{}{}, map[string]interface{}{}, "id")
	assert.Error(t, err)
}

func TestGetCommitMissing(t *testing.T) {
	setMockS3(t, &MockS3{
		err: &types.NoSuchKey{},
	})

	r, _ := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, "id")

	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
}

func TestGetCommitOtherError(t *testing.T) {
	setMockS3(t, &MockS3{
		err: &types.NoSuchBucket{},
	})

	r, _ := remote.Get("s3")
	_, err := r.GetCommit(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, "id")
	assert.Error(t, err)
}

func TestGetCommitMissingMetadata(t *testing.T) {
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]string{},
		},
	})

	r, _ := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, "id")

	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
}

func TestGetCommitBadJson(t *testing.T) {
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]string{"com.dit": "notjson"},
		},
	})

	r, _ := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, "id")

	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
}

func TestGetCommit(t *testing.T) {
	setMockS3(t, &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]string{
				"com.dit": `
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}
`,
			},
		},
	})

	r, _ := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{propBucket: propBucket, propPath: propPath}, map[string]interface{}{}, "id")

	if assert.NoError(t, err) {
		assert.Equal(t, "2019-09-20T13:45:37Z", commit.Properties["timestamp"])
	}
}
