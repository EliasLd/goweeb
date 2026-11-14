package toonkotsuscraper

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"
)

// cryptoFixtures builds key/nonce/URL material in the wire format:
// 32-byte key and 8-byte nonce as unpadded base64url in the fragment.
func cryptoFixtures(t *testing.T) (
	key64, nonce64, pageURL string,
) {
	t.Helper()

	key := bytes.Repeat([]byte{0x37}, 32)
	nonce := []byte("nonenonce")

	key64 = base64.RawURLEncoding.EncodeToString(key)
	nonce64 = base64.RawURLEncoding.EncodeToString(nonce)

	return key64, nonce64,
		"https://toonkotsu.com/cdn/e/abc.bin#k=" + key64 + "&n=" + nonce64
}

// cipherCTR applies the same XOR the site's client uses, so a fixture
// encrypts into plaintext exactly like a real page decrypts.
func cipherCTR(
	t *testing.T,
	key64, nonce64 string,
	data []byte,
) []byte {
	t.Helper()

	key, err := base64.RawURLEncoding.DecodeString(key64)
	if err != nil {
		t.Fatalf("decode fixture key: %v", err)
	}

	nonce, err := base64.RawURLEncoding.DecodeString(nonce64)
	if err != nil {
		t.Fatalf("decode fixture nonce: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("fixture key length: %v", err)
	}

	iv := make([]byte, block.BlockSize())

	copy(iv, nonce)

	out := make([]byte, len(data))

	cipher.NewCTR(block, iv).XORKeyStream(out, data)

	return out
}

func TestDecodeImageRoundTrip(t *testing.T) {
	key64, nonce64, pageURL := cryptoFixtures(t)

	plain := []byte("\xff\xd8\xff\xe0 fake jfif bytes \x00\x01\x02")

	ciphertext := cipherCTR(t, key64, nonce64, plain)

	decoded, ok := DecodeImageData(pageURL, ciphertext)
	if !ok {
		t.Fatal("expected the page to decode")
	}

	if !bytes.Equal(decoded, plain) {
		t.Errorf(
			"decoded = %q, want the original bytes",
			decoded,
		)
	}
}

func TestDecodeImagePassesThroughPagesWithoutKeyMaterial(t *testing.T) {
	cases := []string{
		"https://toonkotsu.com/cdn/c/cover.webp",
		"https://toonkotsu.com/cdn/e/abc.bin#other=x",
		"https://toonkotsu.com/cdn/e/abc.bin",
	}

	for _, pageURL := range cases {
		_, ok := DecodeImageData(pageURL, []byte("anything"))

		if ok {
			t.Errorf(
				"DecodeImageData(%q) claimed the page",
				pageURL,
			)
		}
	}
}

func TestDecodeImageKeepsBytesOnBadKeyMaterial(t *testing.T) {
	cases := []string{
		// Undecodable base64.
		"https://toonkotsu.com/cdn/e/x.bin#k=!!!!&n=nonenonce",
		// Decodes, but never an AES key length.
		"https://toonkotsu.com/cdn/e/x.bin#k=c2hvcnQ&n=nonenonce",
		// Undecodable nonce.
		"https://toonkotsu.com/cdn/e/x.bin#k=key&n=!&",
	}

	const data = "ciphertext"

	for _, pageURL := range cases {
		decoded, ok := DecodeImageData(pageURL, []byte(data))

		if ok {
			t.Errorf(
				"DecodeImageData(%q) claimed the page",
				pageURL,
			)
		}

		if decoded != nil {
			t.Errorf(
				"DecodeImageData(%q) returned bytes",
				pageURL,
			)
		}
	}
}
