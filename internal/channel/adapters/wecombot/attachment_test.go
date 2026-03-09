package wecombot

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"
)

func TestDecryptAttachment(t *testing.T) {
	t.Parallel()

	key := []byte("0123456789abcdef0123456789abcdef")
	plain := []byte("hello wecom")
	encrypted := encryptCBCForTest(t, plain, key)

	got, err := decryptAttachment(encrypted, base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("decrypt attachment: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("unexpected payload: %q", string(got))
	}
}

func encryptCBCForTest(t *testing.T, plain []byte, key []byte) []byte {
	t.Helper()

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	padded := pkcs7PadForTest(plain, aes.BlockSize)
	encrypted := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(encrypted, padded)
	return encrypted
}

func pkcs7PadForTest(raw []byte, blockSize int) []byte {
	padLen := blockSize - (len(raw) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(append([]byte{}, raw...), padding...)
}
