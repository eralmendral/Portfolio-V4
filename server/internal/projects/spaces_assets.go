package projects

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"
	"time"
)

type SpacesAssetStore struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string
	ACL             string
	HTTPClient      *http.Client
	Now             func() time.Time
}

func (s SpacesAssetStore) Validate() error {
	if strings.TrimSpace(s.Bucket) == "" {
		return errors.New("spaces bucket is required")
	}
	if strings.TrimSpace(s.Region) == "" {
		return errors.New("spaces region is required")
	}
	if strings.TrimSpace(s.AccessKeyID) == "" {
		return errors.New("spaces access key id is required")
	}
	if strings.TrimSpace(s.SecretAccessKey) == "" {
		return errors.New("spaces secret access key is required")
	}
	return nil
}

func (s SpacesAssetStore) Put(ctx context.Context, key string, data []byte, contentType string) (StoredAsset, error) {
	if err := s.Validate(); err != nil {
		return StoredAsset{}, err
	}

	request, err := s.newRequest(ctx, http.MethodPut, key, data, contentType)
	if err != nil {
		return StoredAsset{}, err
	}
	if s.ACL != "" {
		request.Header.Set("x-amz-acl", s.ACL)
	}
	s.signRequest(request, data)

	response, err := s.client().Do(request)
	if err != nil {
		return StoredAsset{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return StoredAsset{}, fmt.Errorf("spaces upload failed: %s: %s", response.Status, limitedBody(response.Body))
	}

	return StoredAsset{
		URL:  s.publicURL(key),
		Path: key,
	}, nil
}

func (s SpacesAssetStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := s.Validate(); err != nil {
		return err
	}

	request, err := s.newRequest(ctx, http.MethodDelete, key, nil, "")
	if err != nil {
		return err
	}
	s.signRequest(request, nil)

	response, err := s.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("spaces delete failed: %s: %s", response.Status, limitedBody(response.Body))
	}

	return nil
}

func (s SpacesAssetStore) newRequest(ctx context.Context, method string, key string, data []byte, contentType string) (*http.Request, error) {
	requestURL, err := s.requestURL(key)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if len(data) > 0 {
		body = bytes.NewReader(data)
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	return request, nil
}

func (s SpacesAssetStore) signRequest(request *http.Request, data []byte) {
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}

	payloadHash := sha256Hex(data)
	amzDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")

	request.Host = request.URL.Host
	request.Header.Set("Host", request.URL.Host)
	request.Header.Set("x-amz-content-sha256", payloadHash)
	request.Header.Set("x-amz-date", amzDate)

	canonicalHeaders, signedHeaders := canonicalizeHeaders(request.Header)
	canonicalRequest := strings.Join([]string{
		request.Method,
		request.URL.EscapedPath(),
		request.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{shortDate, s.Region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(signingKey(s.SecretAccessKey, shortDate, s.Region), []byte(stringToSign)))
	request.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.AccessKeyID,
		scope,
		signedHeaders,
		signature,
	))
}

func (s SpacesAssetStore) requestURL(key string) (string, error) {
	endpoint := strings.TrimRight(s.Endpoint, "/")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.digitaloceanspaces.com", s.Region)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	parsed.Host = s.Bucket + "." + parsed.Host
	parsed.Path = "/" + escapeS3Path(key)
	return parsed.String(), nil
}

func (s SpacesAssetStore) publicURL(key string) string {
	baseURL := strings.TrimRight(s.PublicBaseURL, "/")
	if baseURL == "" {
		requestURL, err := s.requestURL(key)
		if err == nil {
			return requestURL
		}
		return key
	}

	return baseURL + "/" + escapeS3Path(key)
}

func (s SpacesAssetStore) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return http.DefaultClient
}

func canonicalizeHeaders(headers http.Header) (string, string) {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, strings.ToLower(name))
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		values := headers.Values(name)
		for i := range values {
			values[i] = strings.Join(strings.Fields(values[i]), " ")
		}
		lines = append(lines, name+":"+strings.Join(values, ",")+"\n")
	}

	return strings.Join(lines, ""), strings.Join(names, ";")
}

func signingKey(secret string, date string, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key []byte, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func escapeS3Path(key string) string {
	segments := strings.Split(pathpkg.Clean("/"+key), "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return strings.TrimPrefix(strings.Join(segments, "/"), "/")
}

func limitedBody(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 4096))
	return strings.TrimSpace(string(data))
}
