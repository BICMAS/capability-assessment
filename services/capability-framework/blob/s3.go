package blob

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Store interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
}

type Memory struct {
	objects map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{objects: map[string][]byte{}}
}

func (m *Memory) Put(_ context.Context, key string, body []byte, _ string) error {
	copied := make([]byte, len(body))
	copy(copied, body)
	m.objects[key] = copied
	return nil
}

func (m *Memory) Get(key string) ([]byte, bool) {
	body, ok := m.objects[key]
	return body, ok
}

func (m *Memory) Objects() map[string][]byte {
	copied := make(map[string][]byte, len(m.objects))
	for key, body := range m.objects {
		copied[key] = body
	}
	return copied
}

type S3 struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
	client    *http.Client
}

func NewS3(endpoint, bucket, accessKey, secretKey string) S3 {
	return S3{
		endpoint:  strings.TrimRight(endpoint, "/"),
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s S3) Put(ctx context.Context, key string, body []byte, contentType string) error {
	parsed, err := url.Parse(s.endpoint)
	if err != nil {
		return fmt.Errorf("parse S3 endpoint: %w", err)
	}
	canonicalURI := "/" + s.bucket + "/" + escapeKey(key)
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256.Sum256(body)
	payloadHex := hex.EncodeToString(payloadHash[:])
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", parsed.Host, payloadHex, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHex,
	}, "\n")
	credentialScope := dateStamp + "/us-east-1/s3/aws4_request"
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hex.EncodeToString(requestHash[:]),
	}, "\n")
	signingKey := hmacSHA256([]byte("AWS4"+s.secretKey), dateStamp)
	signingKey = hmacSHA256(signingKey, "us-east-1")
	signingKey = hmacSHA256(signingKey, "s3")
	signingKey = hmacSHA256(signingKey, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, s.endpoint+canonicalURI, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Host", parsed.Host)
	request.Header.Set("x-amz-content-sha256", payloadHex)
	request.Header.Set("x-amz-date", amzDate)
	request.Header.Set("Authorization", authorization)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("store object: %s: %s", response.Status, strings.TrimSpace(string(payload)))
	}
	return nil
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
