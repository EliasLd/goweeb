package toonkotsuscraper

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// DecodeImageData decrypts one downloaded page when its URL carries
// the site's key material; pages outside the encrypted blob space
// pass through unchanged (false).
//
// Page blobs under /cdn/e/ are AES-CTR ciphertexts: the reader
// payload gives each page a 32-byte key and an 8-byte nonce as
// base64url, and the page step smuggles both back here in the URL
// fragment (#k=…&n=…). Key material that will not decode, or decrypt
// that fails, keeps the original bytes.
func DecodeImageData(pageURL string, data []byte) ([]byte, bool) {
	parsed, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil || parsed.Fragment == "" {
		return nil, false
	}

	params, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		return nil, false
	}

	key := params.Get("k")
	nonce := params.Get("n")

	if key == "" || nonce == "" {
		return nil, false
	}

	plain, err := decryptCTR(key, nonce, data)
	if err != nil {
		return nil, false
	}

	return plain, true
}

// decryptCTR decrypts one page blob with the scheme the site's own
// client uses: the 8-byte nonce left-pads with zero bytes into the
// 16-byte AES-CTR initial block.
func decryptCTR(rawKey, rawNonce string, data []byte) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid toonkotsu page key: %w", err)
	}

	nonce, err := base64.RawURLEncoding.DecodeString(rawNonce)
	if err != nil {
		return nil, fmt.Errorf("invalid toonkotsu page nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("invalid toonkotsu page key: %w", err)
	}

	iv := make([]byte, block.BlockSize())

	copy(iv, nonce)

	plain := make([]byte, len(data))

	cipher.NewCTR(block, iv).XORKeyStream(plain, data)

	return plain, nil
}
