/*
 * Copyright The Titan Project Contributors.
 */
package s3

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/datadatdat/remote-sdk-go/remote"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestRegistered(t *testing.T) {
	r := remote.Get("s3")
	ret, err := r.Type()
	if assert.NoError(t, err) {
		assert.Equal(t, "s3", ret)
	}
}

func TestFromURL(t *testing.T) {
	r := remote.Get("s3")
	props, err := r.FromURL("s3://bucket/object/path", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "bucket", props["bucket"])
		assert.Equal(t, "object/path", props["path"])
		assert.Nil(t, props["accessKey"])
		assert.Nil(t, props["secretKey"])
		assert.Nil(t, props["region"])
	}
}

func TestNoPath(t *testing.T) {
	r := remote.Get("s3")
	props, err := r.FromURL("s3://bucket", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "bucket", props["bucket"])
		assert.Nil(t, props["path"])
		assert.Nil(t, props["accessKey"])
		assert.Nil(t, props["secretKey"])
		assert.Nil(t, props["region"])
	}
}

func TestBadUrl(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://host\nname", map[string]string{})
	assert.Error(t, err)
}

func TestBadScheme(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3", map[string]string{})
	assert.Error(t, err)
}

func TestBadSchemeName(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("foo://bucket/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadProperty(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{"foo": "bar"})
	assert.Error(t, err)
}

func TestBadUser(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://user@bucket/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadUserPassword(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://user:password@bucket/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadPort(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://bucket:80/object/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadMissingBucket(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3:///object/path", map[string]string{})
	assert.Error(t, err)
}

func TestProperties(t *testing.T) {
	r := remote.Get("s3")
	props, err := r.FromURL("s3://bucket/object/path", map[string]string{
		"accessKey": "ACCESS", "secretKey": "SECRET", "region": "REGION",
	})
	if assert.NoError(t, err) {
		assert.Equal(t, "bucket", props["bucket"])
		assert.Equal(t, "object/path", props["path"])
		assert.Equal(t, "ACCESS", props["accessKey"])
		assert.Equal(t, "SECRET", props["secretKey"])
		assert.Equal(t, "REGION", props["region"])
	}
}

func TestBadAccessKeyOnly(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{"accessKey": "ACCESS"})
	assert.Error(t, err)
}

func TestBadSecretKeyOnly(t *testing.T) {
	r := remote.Get("s3")
	_, err := r.FromURL("s3://bucket/object/path", map[string]string{"secretKey": "ACCESS"})
	assert.Error(t, err)
}

func TestToURL(t *testing.T) {
	r := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{"bucket": "bucket", "path": "path"})
	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Empty(t, props)
	}
}

func TestWithKeys(t *testing.T) {
	r := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{"bucket": "bucket", "path": "path",
		"accessKey": "ACCESS", "secretKey": "SECRET"})
	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Len(t, props, 2)
		assert.Equal(t, "ACCESS", props["accessKey"])
		assert.Equal(t, "SECRET", props["secretKey"])
	}
}

func TestWithRegsion(t *testing.T) {
	r := remote.Get("s3")
	u, props, err := r.ToURL(map[string]interface{}{"bucket": "bucket", "path": "path",
		"region": "REGION"})
	if assert.NoError(t, err) {
		assert.Equal(t, "s3://bucket/path", u)
		assert.Len(t, props, 1)
		assert.Equal(t, "REGION", props["region"])
	}
}

func TestGetParameters(t *testing.T) {
	r := remote.Get("s3")
	props, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path",
		"accessKey": "ACCESS", "secretKey": "SECRET", "region": "REGION"})
	if assert.NoError(t, err) {
		assert.Equal(t, "ACCESS", props["accessKey"])
		assert.Equal(t, "SECRET", props["secretKey"])
		assert.Equal(t, "REGION", props["region"])
	}
}

func TestGetParametersEnvironment(t *testing.T) {
	r := remote.Get("s3")
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "ACCESS")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")
	_ = os.Setenv("AWS_REGION", "us-west-2")
	_ = os.Setenv("AWS_SESSION_TOKEN", "TOKEN")
	props, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
	if assert.NoError(t, err) {
		assert.Equal(t, "ACCESS", props["accessKey"])
		assert.Equal(t, "SECRET", props["secretKey"])
		assert.Equal(t, "us-west-2", props["region"])
		assert.Equal(t, "TOKEN", props["sessionToken"])
	}
}

func TestGetParametersFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "s3.test")
	if !assert.NoError(t, err) {
		return
	}
	defer os.RemoveAll(dir)

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
	credContent := `
[default]
aws_access_key_id = ACCESS2
aws_secret_access_key = SECRET2
aws_session_token = TOKEN2
`

	err1 := ioutil.WriteFile(configFile, []byte(configContent), 0600)
	err2 := ioutil.WriteFile(credFile, []byte(credContent), 0600)
	if assert.NoError(t, err1) && assert.NoError(t, err2) {
		_ = os.Setenv("AWS_CONFIG_FILE", configFile)
		_ = os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credFile)

		r := remote.Get("s3")
		props, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
		if assert.NoError(t, err) {
			assert.Equal(t, "ACCESS2", props["accessKey"])
			assert.Equal(t, "SECRET2", props["secretKey"])
			assert.Equal(t, "us-west-1", props["region"])
			assert.Equal(t, "TOKEN2", props["sessionToken"])
		}
	}
}

func TestBadNewSession(t *testing.T) {
	r := remote.Get("s3")
	newSessionWithOptions = func(options session.Options) (session *session.Session, err error) {
		return nil, errors.New("err")
	}
	_, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
	assert.Error(t, err)
	newSessionWithOptions = session.NewSessionWithOptions
}

func TestBadConfigCredentials(t *testing.T) {
	r := remote.Get("s3")
	p := new(MockProvider)
	p.On("Retrieve").Return(credentials.Value{}, errors.New("err"))
	newSessionWithOptions = func(options session.Options) (*session.Session, error) {
		return &session.Session{
			Config: &aws.Config{
				Credentials: credentials.NewCredentials(p),
			},
		}, nil
	}
	_, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
	assert.Error(t, err)
	newSessionWithOptions = session.NewSessionWithOptions
}

func TestBadCredentialsAccessKey(t *testing.T) {
	r := remote.Get("s3")
	p := new(MockProvider)
	p.On("Retrieve").Return(credentials.Value{}, nil)
	newSessionWithOptions = func(options session.Options) (*session.Session, error) {
		return &session.Session{
			Config: &aws.Config{
				Credentials: credentials.NewCredentials(p),
			},
		}, nil
	}
	_, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
	assert.Error(t, err)
	newSessionWithOptions = session.NewSessionWithOptions
}

func TestBadCredentialsRegion(t *testing.T) {
	r := remote.Get("s3")
	p := new(MockProvider)
	p.On("Retrieve").Return(credentials.Value{
		AccessKeyID:     "ACCESS",
		SecretAccessKey: "SECRET",
	}, nil)
	newSessionWithOptions = func(options session.Options) (*session.Session, error) {
		return &session.Session{
			Config: &aws.Config{
				Credentials: credentials.NewCredentials(p),
			},
		}, nil
	}
	_, err := r.GetParameters(map[string]interface{}{"bucket": "bucket", "path": "path"})
	assert.Error(t, err)
	newSessionWithOptions = session.NewSessionWithOptions
}

func TestMetadataKey(t *testing.T) {
	k := getMetadataKey(aws.String("foo"))
	assert.Equal(t, "foo/titan", k)
}

func TestMetadataKeyNil(t *testing.T) {
	k := getMetadataKey(nil)
	assert.Equal(t, "titan", k)
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
	k := getKey(map[string]interface{}{"path": "one/two"}, aws.String("three"))
	assert.Equal(t, "one/two/three", *k)
}

func TestKeyPathNoCommit(t *testing.T) {
	k := getKey(map[string]interface{}{"path": "one/two"}, nil)
	assert.Equal(t, "one/two", *k)
}

func TestValidateRemoteAllProperties(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{"bucket": "bucket", "secretKey": "secret",
		"accessKey": "access", "path": "/path", "region": "region"})
	assert.NoError(t, err)
}

func TestValidateRemoteOnlyRequired(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{"bucket": "bucket"})
	assert.NoError(t, err)
}

func TestValidateRemoteMissingRequired(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{})
	assert.Error(t, err)
}

func TestValidateRemoteInvalidPoperty(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{"bucket": "bucket", "foo": "bar"})
	assert.Error(t, err)
}

func TestValidateRemoteOnlyAccessKey(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{"bucket": "bucket", "accessKey": "access"})
	assert.Error(t, err)
}

func TestValidateRemoteOnlySecretKey(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateRemote(map[string]interface{}{"bucket": "bucket", "secretKey": "secret"})
	assert.Error(t, err)
}

func TestValidateParametersEmpty(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateParameters(map[string]interface{}{})
	assert.NoError(t, err)
}

func TestValidateParametersAll(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateParameters(map[string]interface{}{"accessKey": "access", "secretKey": "secret",
		"region": "region", "sessionToken": "token"})
	assert.NoError(t, err)
}

func TestValidateParametersInvalid(t *testing.T) {
	r := remote.Get("s3")
	err := r.ValidateParameters(map[string]interface{}{"foo": "bar"})
	assert.Error(t, err)
}

var mockConfig *aws.Config

func installMockS3() {
	newSession = func(cfgs ...*aws.Config) (*session.Session, error) {
		mockConfig = cfgs[0]
		return &session.Session{
			Config: mockConfig,
		}, nil
	}
	s3New = func(p client.ConfigProvider, cfgs ...*aws.Config) *s3.S3 {
		return &s3.S3{
			Client: &client.Client{
				Config: *mockConfig,
			},
		}
	}
}

func restoreS3() {
	newSession = session.NewSession
	s3New = s3.New
}

func TestGetS3(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"accessKey": "access", "secretKey": "secret", "region": "region"},
		map[string]interface{}{})
	if assert.NoError(t, err) {
		assert.Equal(t, "region", *mockConfig.Region)
		creds, err := mockConfig.Credentials.Get()
		if assert.NoError(t, err) {
			assert.Equal(t, "access", creds.AccessKeyID)
			assert.Equal(t, "secret", creds.SecretAccessKey)
		}
	}
	restoreS3()
}

func TestGetS3Parameters(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket"},
		map[string]interface{}{"accessKey": "access", "secretKey": "secret", "region": "region", "sessionToken": "token"})
	if assert.NoError(t, err) {
		assert.Equal(t, "region", *mockConfig.Region)
		creds, err := mockConfig.Credentials.Get()
		if assert.NoError(t, err) {
			assert.Equal(t, "access", creds.AccessKeyID)
			assert.Equal(t, "secret", creds.SecretAccessKey)
			assert.Equal(t, "token", creds.SessionToken)
		}
	}
	restoreS3()
}

func TestGetS3MissingRegion(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket"},
		map[string]interface{}{"accessKey": "access", "secretKey": "secret"})
	assert.Error(t, err)
	restoreS3()
}

func TestGetS3MissingAccessKey(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket"},
		map[string]interface{}{"region": "region", "secretKey": "secret"})
	assert.Error(t, err)
	restoreS3()
}

func TestGetS3MissingSecretKey(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket"},
		map[string]interface{}{"region": "region", "accessKey": "access"})
	assert.Error(t, err)
	restoreS3()
}

func TestGetS3BadToken(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket"},
		map[string]interface{}{"accessKey": "access", "secretKey": "secret", "region": "region", "sessionToken": 4})
	assert.Error(t, err)
	restoreS3()
}

func TestGetS3BadRemote(t *testing.T) {
	installMockS3()
	_, err := getS3(map[string]interface{}{"bucket": "bucket", "accessKey": 4},
		map[string]interface{}{"secretKey": "secret", "region": "region"})
	assert.Error(t, err)
	restoreS3()
}

func TestNewSessionFails(t *testing.T) {
	newSession = func(cfgs ...*aws.Config) (*session.Session, error) {
		return nil, errors.New("error")
	}
	_, err := getS3(map[string]interface{}{"accessKey": "access", "secretKey": "secret", "region": "region"},
		map[string]interface{}{})
	assert.Error(t, err)
	newSession = session.NewSession
}

func TestInstallMock(t *testing.T) {
	mockS3 = &MockS3{}
	svc, _ := getS3(map[string]interface{}{}, map[string]interface{}{})
	assert.Equal(t, mockS3, svc)
	mockS3 = nil
}

func TestGetMetadataContent(t *testing.T) {
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader("metadata")),
		},
	}
	res, err := getMetadataContent(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{})
	if assert.NoError(t, err) {
		content, err := ioutil.ReadAll(res)
		if assert.NoError(t, err) {
			assert.Equal(t, "metadata", string(content))
		}

	}

	mockS3 = nil
}

func TestGetMetadataGetS3Error(t *testing.T) {
	installMockS3()
	_, err := getMetadataContent(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{})
	assert.Error(t, err)
	restoreS3()
}

func TestGetMetadataMissing(t *testing.T) {
	mockS3 = &MockS3{
		err: awserr.New(s3.ErrCodeNoSuchKey, "error", nil),
	}
	res, err := getMetadataContent(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{})
	if assert.NoError(t, err) {
		content, err := ioutil.ReadAll(res)
		if assert.NoError(t, err) {
			assert.Equal(t, "", string(content))
		}

	}
	mockS3 = nil
}

func TestGetMetadataOtherError(t *testing.T) {
	mockS3 = &MockS3{
		err: awserr.New(s3.ErrCodeNoSuchBucket, "error", nil),
	}
	_, err := getMetadataContent(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{})
	assert.Error(t, err)
	mockS3 = nil
}

func TestListCommits(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z"}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader(metadata)),
		},
	}
	r := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 2)
		assert.Equal(t, "two", commits[0].Id)
		assert.Equal(t, "one", commits[1].Id)
	}

	mockS3 = nil
}

func TestListCommitsInvalid(t *testing.T) {
	metadata := `
foo
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader(metadata)),
		},
	}
	r := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "two", commits[0].Id)
	}

	mockS3 = nil
}

func TestListCommitsTags(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z", "tags": { "a": "b" }}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}`
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader(metadata)),
		},
	}
	r := remote.Get("s3")
	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "one", commits[0].Id)
	}
	mockS3 = nil
}

func TestListCommitsError(t *testing.T) {
	mockS3 = &MockS3{
		err: errors.New("error"),
	}
	r := remote.Get("s3")
	_, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})
	assert.Error(t, err)
	mockS3 = nil
}

func TestGetCommitBadS3(t *testing.T) {
	installMockS3()
	r := remote.Get("s3")
	_, err := r.GetCommit(map[string]interface{}{}, map[string]interface{}{}, "id")
	assert.Error(t, err)
	restoreS3()
}

func TestGetCommitMissing(t *testing.T) {
	mockS3 = &MockS3{
		err: awserr.New(s3.ErrCodeNoSuchKey, "error", nil),
	}
	r := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "id")
	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
	mockS3 = nil
}

func TestGetCommitOtherError(t *testing.T) {
	mockS3 = &MockS3{
		err: awserr.New(s3.ErrCodeNoSuchBucket, "error", nil),
	}
	r := remote.Get("s3")
	_, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "id")
	assert.Error(t, err)
	mockS3 = nil
}

func TestGetCommitMissingMetadata(t *testing.T) {
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]*string{},
		},
	}
	r := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "id")
	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
	mockS3 = nil
}

func TestGetCommitBadJson(t *testing.T) {
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]*string{"io.titan-data": aws.String("notjson")},
		},
	}
	r := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "id")
	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}
	mockS3 = nil
}

func TestGetCommit(t *testing.T) {
	mockS3 = &MockS3{
		GetObjectOutput: s3.GetObjectOutput{
			Metadata: map[string]*string{"io.titan-data": aws.String(`
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}
`)},
		},
	}
	r := remote.Get("s3")
	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "id")
	if assert.NoError(t, err) {
		assert.Equal(t, "2019-09-20T13:45:37Z", commit.Properties["timestamp"])
	}
	mockS3 = nil
}
