package sku

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultPrefix   = "MS"
	randomChunkSize = 6
)

var charset = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func GenerateInternal() string {
	return GenerateInternalWithPrefix(DefaultPrefix)
}

func GenerateInternalWithPrefix(prefix string) string {
	normalizedPrefix := strings.ToUpper(strings.TrimSpace(prefix))
	if normalizedPrefix == "" {
		normalizedPrefix = DefaultPrefix
	}
	datePart := time.Now().UTC().Format("20060102")
	randomPart := randomChunk(randomChunkSize)
	return fmt.Sprintf("%s-%s-%s", normalizedPrefix, datePart, randomPart)
}

func randomChunk(length int) string {
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		fallback := strings.ToUpper(fmt.Sprintf("%x", time.Now().UTC().UnixNano()))
		if len(fallback) >= length {
			return fallback[:length]
		}
		return fallback
	}

	chunk := make([]byte, length)
	for i := range randomBytes {
		chunk[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(chunk)
}
