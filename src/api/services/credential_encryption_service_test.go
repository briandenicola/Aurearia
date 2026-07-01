package services

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testCredentialKey() string {
	return base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
}

func TestCredentialEncryptionService_RoundTrip(t *testing.T) {
	svc, err := NewCredentialEncryptionService(testCredentialKey())
	if err != nil {
		t.Fatalf("NewCredentialEncryptionService: %v", err)
	}

	encrypted, err := svc.EncryptStringWithAAD("secret-password", []byte("user:1:numis"))
	if err != nil {
		t.Fatalf("EncryptStringWithAAD: %v", err)
	}
	if encrypted == "secret-password" {
		t.Fatal("encrypted credential matched plaintext")
	}
	if !strings.HasPrefix(encrypted, encryptedCredentialPrefix) {
		t.Fatalf("encrypted credential prefix = %q", encrypted)
	}

	plain, wasEncrypted, err := svc.DecryptStringWithAAD(encrypted, []byte("user:1:numis"))
	if err != nil {
		t.Fatalf("DecryptStringWithAAD: %v", err)
	}
	if !wasEncrypted {
		t.Fatal("DecryptStringWithAAD reported plaintext for encrypted value")
	}
	if plain != "secret-password" {
		t.Fatalf("decrypted credential = %q", plain)
	}
}

func TestCredentialEncryptionService_LegacyPlaintext(t *testing.T) {
	svc, err := NewCredentialEncryptionService(testCredentialKey())
	if err != nil {
		t.Fatalf("NewCredentialEncryptionService: %v", err)
	}

	plain, wasEncrypted, err := svc.DecryptStringWithAAD("legacy-password", []byte("user:1:cng"))
	if err != nil {
		t.Fatalf("DecryptStringWithAAD legacy: %v", err)
	}
	if wasEncrypted {
		t.Fatal("legacy plaintext reported as encrypted")
	}
	if plain != "legacy-password" {
		t.Fatalf("legacy plaintext = %q", plain)
	}
}

func TestCredentialEncryptionService_RejectsTamperWrongKeyAndWrongAAD(t *testing.T) {
	svc, err := NewCredentialEncryptionService(testCredentialKey())
	if err != nil {
		t.Fatalf("NewCredentialEncryptionService: %v", err)
	}
	encrypted, err := svc.EncryptStringWithAAD("secret-password", []byte("user:1:numis"))
	if err != nil {
		t.Fatalf("EncryptStringWithAAD: %v", err)
	}

	if _, _, err := svc.DecryptStringWithAAD(encrypted, []byte("user:1:cng")); err == nil {
		t.Fatal("DecryptStringWithAAD succeeded with wrong AAD")
	}

	wrongKeySvc, err := NewCredentialEncryptionService(base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789")))
	if err != nil {
		t.Fatalf("wrong key service: %v", err)
	}
	if _, _, err := wrongKeySvc.DecryptStringWithAAD(encrypted, []byte("user:1:numis")); err == nil {
		t.Fatal("DecryptStringWithAAD succeeded with wrong key")
	}

	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encrypted, encryptedCredentialPrefix))
	if err != nil {
		t.Fatalf("decode encrypted payload for tamper test: %v", err)
	}
	payload[len(payload)-1] ^= 0xff
	tampered := encryptedCredentialPrefix + base64.RawURLEncoding.EncodeToString(payload)
	if _, _, err := svc.DecryptStringWithAAD(tampered, []byte("user:1:numis")); err == nil {
		t.Fatal("DecryptStringWithAAD succeeded with tampered ciphertext")
	}
}

func TestCredentialEncryptionService_KeyValidation(t *testing.T) {
	if _, err := NewCredentialEncryptionService("too-short"); err == nil {
		t.Fatal("NewCredentialEncryptionService accepted short key")
	}
	if _, err := NewCredentialEncryptionService(testCredentialKey()); err != nil {
		t.Fatalf("NewCredentialEncryptionService rejected base64 32-byte key: %v", err)
	}
}
