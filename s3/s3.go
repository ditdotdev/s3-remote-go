/*
 * Copyright The Titan Project Contributors.
 */
package s3

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/datadatdat/remote-sdk-go/remote"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
)

/*
 * The S3 provider is a very simple provider for storing whole commits directly in a S3 bucket. Each commit is is a
 * key within a folder, for example:
 *
 *      s3://bucket/path/to/repo/3583-4053-598ea-298fa
 *
 * Within each commit sub-directory, there is .tar.gz file for each volume. The metadata for each commit is stored
 * as metadata for the object, as well in a 'titan' file at the root of the repository, with once line per commit. We
 * do this for a few reasons:
 *
 *      * Storing it in object metadata is inefficient, as there's no way to fetch the metadata of multiple objects
 *        at once. We keep it per-object for the cases where we
 *      * We want to be able to access this data in a read-only fashion over the HTTP interface, and there is no way
 *        to access object metadata (or even necessarily iterate over objects) through the HTTP interface.
 *
 * This has its downsides, namely that deleting a commit is more complicated, and there is greater risk of
 * concurrent operations creating invalid state, but those are existing challenges with these simplistic providers.
 * Properly solving them would require a more sophisticated provider with server-side logic.
 *
 * The URI syntax for S3 remotes is:
 *
 *      s3://bucket[/object]
 *
 * The following properties are supported:
 *
 *      accessKey       AWS access key.
 *
 *      secretKey       AWS secret key.
 *
 *      region          AWS region.
 *
 * While all of these can be specified in the remote, best practices are to leave them blank, and have them pulled
 * from the user's environment at the time the operation request is made.
 */

type s3Remote struct {
}

const metadataProperty = "io.titan-data"

func (s s3Remote) Type() (string, error) {
	return "s3", nil
}

func (s s3Remote) FromURL(rawUrl string, additionalProperties map[string]string) (map[string]interface{}, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "s3" {
		return nil, errors.New("invalid remote scheme")
	}

	if u.User != nil {
		return nil, errors.New("remote username and password cannot be specified")
	}

	if u.Port() != "" {
		return nil, errors.New("remote port cannot be specified")
	}

	if u.Hostname() == "" {
		return nil, errors.New("missing remote bucket name")
	}

	accessKey := additionalProperties["accessKey"]
	secretKey := additionalProperties["secretKey"]
	region := additionalProperties["region"]
	for k := range additionalProperties {
		if k != "accessKey" && k != "secretKey" && k != "region" {
			return nil, fmt.Errorf("invalid remote property '%s'", k)
		}
	}

	if (accessKey == "" && secretKey != "") || (accessKey != "" && secretKey == "") {
		return nil, fmt.Errorf("either both of accessKey and secretKey must be set, or neither")
	}

	path := u.Path
	if strings.Index(path, "/") == 0 {
		path = path[1:]
	}

	result := map[string]interface{}{"bucket": u.Hostname()}
	if accessKey != "" {
		result["accessKey"] = accessKey
	}
	if secretKey != "" {
		result["secretKey"] = secretKey
	}
	if region != "" {
		result["region"] = region
	}
	if path != "" {
		result["path"] = path
	}

	return result, nil
}

func (s s3Remote) ToURL(properties map[string]interface{}) (string, map[string]string, error) {
	u := fmt.Sprintf("s3://%s", properties["bucket"])
	if properties["path"] != nil {
		u += fmt.Sprintf("/%s", properties["path"])
	}

	params := map[string]string{}
	if properties["accessKey"] != nil {
		params["accessKey"] = properties["accessKey"].(string)
	}
	if properties["secretKey"] != nil {
		params["secretKey"] = properties["secretKey"].(string)
	}
	if properties["region"] != nil {
		params["region"] = properties["region"].(string)
	}

	return u, params, nil
}

// AWS SDK methods visible for testing
var newSessionWithOptions = session.NewSessionWithOptions
var newSession = session.NewSession
var s3New = s3.New
var mockS3 s3iface.S3API

func (s s3Remote) GetParameters(remoteProperties map[string]interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if remoteProperties["accessKey"] != nil {
		result["accessKey"] = remoteProperties["accessKey"].(string)
	}
	if remoteProperties["secretKey"] != nil {
		result["secretKey"] = remoteProperties["secretKey"].(string)
	}
	if remoteProperties["region"] != nil {
		result["region"] = remoteProperties["region"].(string)
	}

	if result["accessKey"] == nil || result["secretKey"] == nil || result["region"] == nil {
		sess, err := newSessionWithOptions(session.Options{SharedConfigState: session.SharedConfigEnable})
		if err != nil {
			return nil, err
		}

		creds, err := sess.Config.Credentials.Get()
		if err != nil {
			return nil, err
		}

		if result["accessKey"] == nil && creds.AccessKeyID != "" {
			result["accessKey"] = creds.AccessKeyID
		}
		if result["secretKey"] == nil && creds.SecretAccessKey != "" {
			result["secretKey"] = creds.SecretAccessKey
		}
		if creds.SessionToken != "" {
			result["sessionToken"] = creds.SessionToken
		}
		if result["region"] == nil && sess.Config.Region != nil {
			result["region"] = *sess.Config.Region
		}

		if result["accessKey"] == nil || result["secretKey"] == nil {
			return nil, errors.New("unable to determine AWS credentials")
		}
		if result["region"] == nil {
			return nil, errors.New("unable to determine AWS region")
		}
	}

	return result, nil
}

/*
 * Validate a S3 remote. The only required field is "bucket". Optional fields include (path, accessKey,
 * secretKey, region). If either accessKey or secretKey is specified, then both must be specified.
 */
func (s s3Remote) ValidateRemote(properties map[string]interface{}) error {
	err := remote.ValidateFields(properties, []string{"bucket"}, []string{"path", "accessKey", "secretKey", "region"})
	if err != nil {
		return err
	}

	_, hasAccess := properties["accessKey"]
	_, hasSecret := properties["secretKey"]
	if (hasAccess && !hasSecret) || (!hasAccess && hasSecret) {
		return fmt.Errorf("either both of accessKey and secretKey must be set, or neither")
	}

	return nil
}

/*
 * Validate S3 parameters. All parameters are optional: (accessKey, secretKey, region, sessionToken).
 */
func (s s3Remote) ValidateParameters(parameters map[string]interface{}) error {
	return remote.ValidateFields(parameters, []string{}, []string{"accessKey", "secretKey", "region", "sessionToken"})
}

func getRemoteValue(remote map[string]interface{}, parameters map[string]interface{}, field string) (string, error) {
	if raw, ok := parameters[field]; ok {
		if value, ok := raw.(string); ok {
			return value, nil
		} else {
			return "", fmt.Errorf("invalid parameter, '%s' must be a string", field)
		}
	}

	if remote == nil {
		return "", nil
	}

	if raw, ok := remote[field]; ok {
		if value, ok := raw.(string); ok {
			return value, nil
		} else {
			return "", fmt.Errorf("invalid parameter, '%s' must be a string", field)
		}
	}
	return "", fmt.Errorf("missing parameter '%s'", field)
}

/*
 * Get an instance of the S3 service based on the remote configuration and parameters.
 */
func getS3(remote map[string]interface{}, parameters map[string]interface{}) (s3iface.S3API, error) {
	if mockS3 != nil {
		return mockS3, nil
	}

	accessKey, err := getRemoteValue(remote, parameters, "accessKey")
	if err != nil {
		return nil, err
	}

	secretKey, err := getRemoteValue(remote, parameters, "secretKey")
	if err != nil {
		return nil, err
	}

	region, err := getRemoteValue(remote, parameters, "region")
	if err != nil {
		return nil, err
	}

	sessionToken, err := getRemoteValue(nil, parameters, "sessionToken")
	if err != nil {
		return nil, err
	}

	sess, err := newSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, sessionToken),
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return s3New(sess), nil
}

/*
 * This function will return the key that identifies the given commit (or root key if no commit
 * is specified). This takes into the account the optional path configured in the remote. Public for testing.
 */
func getKey(remote map[string]interface{}, commitId *string) *string {
	if _, ok := remote["path"]; !ok {
		return commitId
	}
	path := remote["path"].(string)
	if commitId == nil {
		return &path
	}
	res := fmt.Sprintf("%s/%s", path, *commitId)
	return &res
}

/*
 * Gets the path to the titan repo metadata file, which is either in the root of the bucket (if the path is
 * null) or within the path directory.
 */
func getMetadataKey(path *string) string {
	if path == nil {
		return "titan"
	} else {
		return fmt.Sprintf("%s/titan", *path)
	}
}

/*
 * Helper function that fetches the content of the metadata file as an input stream. Returns an empty file if
 * it doesn't yet exist.
 */
func getMetadataContent(remote map[string]interface{}, parameters map[string]interface{}) (io.ReadCloser, error) {
	svc, err := getS3(remote, parameters)
	if err != nil {
		return nil, err
	}
	key := getKey(remote, nil)
	bucket := remote["bucket"].(string)

	req := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    aws.String(getMetadataKey(key)),
	}
	res, err := svc.GetObject(&req)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchKey {
				return ioutil.NopCloser(strings.NewReader("")), nil
			}
		}
		return nil, err
	}
	return res.Body, nil
}

type MetadataCommit struct {
	Id         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

/*
 * List all commits in a repository. This operates by processing the metadata file at the root of the S3 path. Each
 * line is a JSON object with an "id" field and "properties" field.
 */
func (s s3Remote) ListCommits(properties map[string]interface{}, parameters map[string]interface{}, tags []remote.Tag) ([]remote.Commit, error) {
	var ret []remote.Commit
	metadata, err := getMetadataContent(properties, parameters)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(metadata)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if (line) != "" {
			commit := MetadataCommit{}
			err = json.Unmarshal([]byte(line), &commit)
			if err == nil && commit.Properties != nil && commit.Id != "" && remote.MatchTags(commit.Properties, tags) {
				ret = append(ret, remote.Commit{Id: commit.Id, Properties: commit.Properties})
			}
		}
	}

	remote.SortCommits(ret)

	return ret, nil
}

/*
 * Get the metadata for a single commit. This is stored as a user property on the object with the key
 * "io.titan-data". For historical reasons, we keep the metadata within the "properties" sub-object. This
 * matches how it's stored in the top-level metadata file.
 */
func (s s3Remote) GetCommit(properties map[string]interface{}, parameters map[string]interface{}, commitId string) (*remote.Commit, error) {
	svc, err := getS3(properties, parameters)
	if err != nil {
		return nil, err
	}
	key := getKey(properties, &commitId)
	bucket := properties["bucket"].(string)

	req := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    key,
	}
	res, err := svc.GetObject(&req)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == s3.ErrCodeNoSuchKey {
				return nil, nil
			}
		}
		return nil, err
	}

	metadata, ok := res.Metadata[metadataProperty]
	if !ok || metadata == nil {
		return nil, nil
	}

	commit := MetadataCommit{}
	err = json.Unmarshal([]byte(*metadata), &commit)
	if err != nil {
		return nil, nil
	}

	nativeCommit := remote.Commit{Id: commit.Id, Properties: commit.Properties}
	return &nativeCommit, nil
}

func init() {
	remote.Register(s3Remote{})
}
