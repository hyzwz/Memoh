package wecom

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDecodeAESKey(t *testing.T) {
	// WeCom EncodingAESKey is 43-char base64-encoded, decodes to 32 bytes.
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	key, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatalf("DecodeAESKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

func TestDecodeAESKeyInvalid(t *testing.T) {
	_, err := DecodeAESKey("tooshort")
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestSignature(t *testing.T) {
	token := "testtoken"
	timestamp := "1409659813"
	nonce := "1372623149"
	encrypt := "msg_encrypt_content"

	sig := CalcSignature(token, timestamp, nonce, encrypt)
	if sig == "" {
		t.Fatal("signature should not be empty")
	}
	// Signature should be 40 hex chars (SHA1).
	if len(sig) != 40 {
		t.Fatalf("signature length = %d, want 40", len(sig))
	}

	// Same inputs should produce same signature.
	sig2 := CalcSignature(token, timestamp, nonce, encrypt)
	if sig != sig2 {
		t.Fatal("signature should be deterministic")
	}

	// Different input should produce different signature.
	sig3 := CalcSignature(token, timestamp, nonce, "different")
	if sig == sig3 {
		t.Fatal("different input should produce different signature")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Generate a valid 32-byte AES key.
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, err := DecodeAESKey(encodingAESKey)
	if err != nil {
		t.Fatal(err)
	}
	corpID := "wx1234567890"
	plaintext := "Hello, WeCom!"

	encrypted, err := Encrypt(aesKey, plaintext, corpID)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encrypted == "" {
		t.Fatal("encrypted should not be empty")
	}

	// Should be valid base64.
	_, err = base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("encrypted should be base64: %v", err)
	}

	// Decrypt.
	decrypted, receivedCorpID, err := Decrypt(aesKey, encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
	if receivedCorpID != corpID {
		t.Fatalf("corpID = %q, want %q", receivedCorpID, corpID)
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	aesKey := make([]byte, 32)
	_, _, err := Decrypt(aesKey, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, _ := DecodeAESKey(encodingAESKey)
	corpID := "wx1234567890"

	encrypted, err := Encrypt(aesKey, "secret message", corpID)
	if err != nil {
		t.Fatal(err)
	}

	// Try with different key.
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i)
	}
	_, _, err = Decrypt(wrongKey, encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncryptLargeMessage(t *testing.T) {
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	aesKey, _ := DecodeAESKey(encodingAESKey)
	corpID := "wx1234567890"

	largeMsg := strings.Repeat("A", 10000)
	encrypted, err := Encrypt(aesKey, largeMsg, corpID)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, _, err := Decrypt(aesKey, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != largeMsg {
		t.Fatalf("large message roundtrip failed: got %d chars, want %d", len(decrypted), len(largeMsg))
	}
}
