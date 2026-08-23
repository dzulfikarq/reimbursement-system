package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Signed double-submit pattern (docs/06): the cookie carries
// "<nonce>.<hmac(nonce)>" and is readable by JS; mutations must echo it in the
// X-CSRF-Token header. Cross-site attackers cannot read the cookie to set the
// header, and the HMAC stops nonce forgery. Stateless — no server-side store.

var ErrMismatch = errors.New("csrf token mismatch")

func Issue(secret string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	mac := hmacMAC(secret, nonce)
	return hex.EncodeToString(nonce) + "." + hex.EncodeToString(mac), nil
}

// Verify checks header value against cookie value structurally + HMAC.
func Verify(secret, cookieVal, headerVal string) error {
	if cookieVal == "" || headerVal == "" || cookieVal != headerVal {
		return ErrMismatch
	}
	parts := split(cookieVal)
	if len(parts) != 2 {
		return ErrMismatch
	}
	nonce, err := hex.DecodeString(parts[0])
	if err != nil || len(nonce) == 0 {
		return ErrMismatch
	}
	want := hmacMAC(secret, nonce)
	got, err := hex.DecodeString(parts[1])
	if err != nil || !hmac.Equal(want, got) {
		return ErrMismatch
	}
	return nil
}

func split(v string) []string {
	out := make([]string, 0, 2)
	start := 0
	for i := 0; i < len(v); i++ {
		if v[i] == '.' {
			out = append(out, v[start:i])
			start = i + 1
		}
	}
	out = append(out, v[start:])
	return out
}

func hmacMAC(secret string, data []byte) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return h.Sum(nil)
}
