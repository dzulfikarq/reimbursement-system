// Package upload validates uploaded files: size cap + MIME allowlist checked
// by magic bytes, never by client-declared Content-Type (AGENTS.md #4).
package upload

import (
	"bytes"
	"errors"
	"io"
)

const MaxSize = 5 << 20 // 5 MB

var (
	ErrTooLarge   = errors.New("file exceeds 5 MB limit")
	ErrEmpty      = errors.New("file is empty")
	ErrMIME       = errors.New("unsupported file type (allowed: png, jpeg, webp, pdf)")
	ErrReadFailed = errors.New("could not read uploaded file")
)

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
var jpegMagic = []byte{0xFF, 0xD8, 0xFF}
var riffMagic = []byte("RIFF")
var webpMagic = []byte("WEBP")
var pdfMagic = []byte("%PDF-")

// Sniff returns the canonical MIME type from magic bytes.
func Sniff(head []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(head, pngMagic):
		return "image/png", true
	case bytes.HasPrefix(head, jpegMagic):
		return "image/jpeg", true
	case len(head) >= 12 && bytes.HasPrefix(head, riffMagic) && bytes.Equal(head[8:12], webpMagic):
		return "image/webp", true
	case bytes.HasPrefix(head, pdfMagic):
		return "application/pdf", true
	default:
		return "", false
	}
}

// Validate reads the whole body enforcing size + magic-byte type. Returns the
// verified content and detected MIME so callers store exactly what was
// checked.
func Validate(r io.Reader) ([]byte, string, error) {
	content, err := io.ReadAll(io.LimitReader(r, MaxSize+1))
	if err != nil {
		return nil, "", ErrReadFailed
	}
	if len(content) == 0 {
		return nil, "", ErrEmpty
	}
	if int64(len(content)) > MaxSize {
		return nil, "", ErrTooLarge
	}
	mime, ok := Sniff(content)
	if !ok {
		return nil, "", ErrMIME
	}
	return content, mime, nil
}
