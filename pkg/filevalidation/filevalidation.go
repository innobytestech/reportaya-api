// Package filevalidation provides file upload validation utilities:
// MIME type detection via magic bytes, file size validation, filename sanitization,
// and extension allowlisting.
package filevalidation

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// nonSafeCharsRe matches characters that are not safe for filenames.
var nonSafeCharsRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// DetectMIMEType detects the real MIME type of the file by reading magic bytes (first 512 bytes).
// It does NOT trust the client-provided Content-Type header.
func DetectMIMEType(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open file for sniffing: %w", err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read file for sniffing: %w", err)
	}
	if n == 0 {
		return "application/octet-stream", nil
	}

	detected := http.DetectContentType(buf[:n])

	// http.DetectContentType returns "application/octet-stream" for WebP.
	// Check WebP magic bytes manually: RIFF????WEBP
	if detected == "application/octet-stream" && n >= 12 {
		if string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
			return "image/webp", nil
		}
	}

	return detected, nil
}

// ValidateFileSize checks that the file does not exceed maxMB megabytes.
func ValidateFileSize(fh *multipart.FileHeader, maxMB int64) error {
	if maxMB <= 0 {
		maxMB = 10
	}
	if fh.Size > maxMB*1024*1024 {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("file too large, max %dMB", maxMB))
	}
	return nil
}

// SanitizeFilename strips path components, removes unsafe characters, limits length,
// and falls back to a UUID name if the result is empty.
func SanitizeFilename(name string) string {
	// Strip any directory components (path traversal prevention)
	base := filepath.Base(name)
	if base == "." || base == "/" || base == "\\" {
		base = ""
	}

	// Replace unsafe characters
	safe := nonSafeCharsRe.ReplaceAllString(base, "_")

	// Remove leading dots (hidden files / parent-dir tricks)
	safe = strings.TrimLeft(safe, "._")

	// Limit length
	if len(safe) > 255 {
		ext := filepath.Ext(safe)
		nameOnly := safe[:255-len(ext)]
		safe = nameOnly + ext
	}

	// Fallback if empty
	if safe == "" || safe == "." {
		safe = uuid.NewString()
	}

	return safe
}

// SanitizeFolder cleans a storage key prefix supplied by the client. It preserves
// legitimate slash-separated segments (e.g. "products/images") but strips traversal
// sequences (".."), leading/trailing slashes, unsafe characters, and empty segments.
// Falls back to "general" when nothing safe remains.
func SanitizeFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.ReplaceAll(folder, "\\", "/")

	var segments []string
	for _, seg := range strings.Split(folder, "/") {
		seg = strings.TrimSpace(seg)
		// Drop empty segments (collapses // and leading/trailing slashes) and
		// any segment that is "." or contains a parent-dir reference.
		if seg == "" || seg == "." || strings.Contains(seg, "..") {
			continue
		}
		seg = nonSafeCharsRe.ReplaceAllString(seg, "_")
		seg = strings.TrimLeft(seg, "._")
		if seg == "" {
			continue
		}
		segments = append(segments, seg)
	}

	if len(segments) == 0 {
		return "general"
	}
	return strings.Join(segments, "/")
}

// ValidateExtension checks that the file extension (case-insensitive) is in the allowed list.
// allowed entries should include the dot, e.g. []string{".jpg", ".png"}.
func ValidateExtension(filename string, allowed []string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fiber.NewError(fiber.StatusBadRequest, "file has no extension")
	}
	for _, a := range allowed {
		if ext == strings.ToLower(a) {
			return nil
		}
	}
	return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("file extension %q not allowed", ext))
}

// Common extension allowlists for reuse.
var (
	ImageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}
	AllExtensions   = []string{".jpg", ".jpeg", ".png", ".webp", ".pdf", ".xml"}
)
