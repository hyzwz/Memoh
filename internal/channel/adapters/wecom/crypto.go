package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// DecodeAESKey decodes WeCom's 43-char EncodingAESKey to a 32-byte AES key.
func DecodeAESKey(encodingAESKey string) ([]byte, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("invalid EncodingAESKey length: %d, expected 43", len(encodingAESKey))
	}
	return base64.StdEncoding.DecodeString(encodingAESKey + "=")
}

// CalcSignature computes the SHA1 signature used by WeCom message verification.
func CalcSignature(token, timestamp, nonce, encrypt string) string {
	params := []string{token, timestamp, nonce, encrypt}
	sort.Strings(params)
	h := sha1.New()
	io.WriteString(h, strings.Join(params, ""))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Encrypt encrypts plaintext using AES-256-CBC for WeCom.
// Format: random(16) + msgLen(4, network order) + msg + corpID, then PKCS7 padded.
func Encrypt(aesKey []byte, plaintext, corpID string) (string, error) {
	// 16 bytes random
	random := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return "", err
	}

	msgBytes := []byte(plaintext)
	corpIDBytes := []byte(corpID)

	// Build: random(16) + msgLen(4) + msg + corpID
	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(msgBytes)))

	buf := make([]byte, 0, 16+4+len(msgBytes)+len(corpIDBytes))
	buf = append(buf, random...)
	buf = append(buf, msgLen...)
	buf = append(buf, msgBytes...)
	buf = append(buf, corpIDBytes...)

	// PKCS7 padding
	buf = pkcs7Pad(buf, aes.BlockSize)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	// IV is the first 16 bytes of the AES key
	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(buf, buf)

	return base64.StdEncoding.EncodeToString(buf), nil
}

// Decrypt decrypts a WeCom encrypted message. Returns (plaintext, corpID, error).
func Decrypt(aesKey []byte, ciphertext string) (string, string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", "", fmt.Errorf("base64 decode: %w", err)
	}

	if len(cipherBytes) < aes.BlockSize || len(cipherBytes)%aes.BlockSize != 0 {
		return "", "", errors.New("invalid ciphertext length")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", "", err
	}

	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherBytes, cipherBytes)

	// Remove PKCS7 padding
	plainBytes, err := pkcs7Unpad(cipherBytes)
	if err != nil {
		return "", "", err
	}

	// Format: random(16) + msgLen(4) + msg + corpID
	if len(plainBytes) < 20 {
		return "", "", errors.New("decrypted data too short")
	}

	msgLen := binary.BigEndian.Uint32(plainBytes[16:20])
	if 20+msgLen > uint32(len(plainBytes)) {
		return "", "", errors.New("invalid message length in decrypted data")
	}

	msg := string(plainBytes[20 : 20+msgLen])
	corpID := string(plainBytes[20+msgLen:])

	return msg, corpID, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padBytes := make([]byte, padding)
	for i := range padBytes {
		padBytes[i] = byte(padding)
	}
	return append(data, padBytes...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-padding], nil
}
