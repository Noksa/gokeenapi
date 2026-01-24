package gokeencache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainValidationCache(t *testing.T) {
	domain := "example.com"

	// Initially not cached
	_, cached := GetDomainValidation(domain)
	assert.False(t, cached)

	// Cache as valid
	SetDomainValidation(domain, true)

	// Should be cached now
	valid, cached := GetDomainValidation(domain)
	assert.True(t, cached)
	assert.True(t, valid)

	// Cache another domain as invalid
	invalidDomain := "invalid..domain"
	SetDomainValidation(invalidDomain, false)

	valid, cached = GetDomainValidation(invalidDomain)
	assert.True(t, cached)
	assert.False(t, valid)
}

func TestURLContentWithChecksum(t *testing.T) {
	url := "https://example.com/domains.txt"
	content := "example.com\ntest.com\n"
	ttl := 5 * time.Minute

	// Set content
	SetURLContent(url, content, ttl)

	// Get content back
	retrieved, ok := GetURLContent(url)
	require.True(t, ok)
	assert.Equal(t, content, retrieved)

	// Get checksum
	checksum := GetURLChecksum(url)
	assert.NotEmpty(t, checksum)

	// Update content with different data
	newContent := "example.com\ntest.com\nnew.com\n"
	SetURLContent(url, newContent, ttl)

	// Checksum should be different
	newChecksum := GetURLChecksum(url)
	assert.NotEmpty(t, newChecksum)
	assert.NotEqual(t, checksum, newChecksum)
}

func TestComputeChecksum(t *testing.T) {
	content1 := []byte("example.com\ntest.com\n")
	content2 := []byte("example.com\ntest.com\n")
	content3 := []byte("different.com\n")

	checksum1 := ComputeChecksum(content1)
	checksum2 := ComputeChecksum(content2)
	checksum3 := ComputeChecksum(content3)

	// Same content should produce same checksum
	assert.Equal(t, checksum1, checksum2)

	// Different content should produce different checksum
	assert.NotEqual(t, checksum1, checksum3)
}
