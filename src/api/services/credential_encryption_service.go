package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedCredentialPrefix = "enc:v1:"

// CredentialEncryptionService encrypts stored external-provider credentials.
type CredentialEncryptionService struct {
	aead cipher.AEAD
}

// NewCredentialEncryptionService creates an AES-GCM credential encryption service.
func NewCredentialEncryptionService(rawKey string) (*CredentialEncryptionService, error) {
	key, err := parseCredentialEncryptionKey(rawKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create credential cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create credential AEAD: %w", err)
	}
	return &CredentialEncryptionService{aead: aead}, nil
}

// NewDisabledCredentialEncryptionService preserves plaintext behavior when no key is configured in development.
func NewDisabledCredentialEncryptionService() *CredentialEncryptionService {
	return &CredentialEncryptionService{}
}

func (s *CredentialEncryptionService) Enabled() bool {
	return s != nil && s.aead != nil
}

func (s *CredentialEncryptionService) IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedCredentialPrefix)
}

func (s *CredentialEncryptionService) EncryptString(plain string) (string, error) {
	return s.EncryptStringWithAAD(plain, nil)
}

func (s *CredentialEncryptionService) EncryptStringWithAAD(plain string, aad []byte) (string, error) {
	if plain == "" || !s.Enabled() {
		return plain, nil
	}
	if s.IsEncrypted(plain) {
		return plain, nil
	}

	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate credential nonce: %w", err)
	}
	ciphertext := s.aead.Seal(nil, nonce, []byte(plain), aad)
	encoded := make([]byte, 0, len(nonce)+len(ciphertext))
	encoded = append(encoded, nonce...)
	encoded = append(encoded, ciphertext...)
	return encryptedCredentialPrefix + base64.RawURLEncoding.EncodeToString(encoded), nil
}

func (s *CredentialEncryptionService) DecryptString(stored string) (plain string, wasEncrypted bool, err error) {
	return s.DecryptStringWithAAD(stored, nil)
}

func (s *CredentialEncryptionService) DecryptStringWithAAD(stored string, aad []byte) (plain string, wasEncrypted bool, err error) {
	if stored == "" {
		return "", false, nil
	}
	if !s.IsEncrypted(stored) {
		return stored, false, nil
	}
	if !s.Enabled() {
		return "", true, fmt.Errorf("credential encryption key is not configured")
	}

	raw := strings.TrimPrefix(stored, encryptedCredentialPrefix)
	combined, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", true, fmt.Errorf("decode encrypted credential: %w", err)
	}
	nonceSize := s.aead.NonceSize()
	if len(combined) <= nonceSize {
		return "", true, fmt.Errorf("encrypted credential payload is too short")
	}
	nonce := combined[:nonceSize]
	ciphertext := combined[nonceSize:]
	decrypted, err := s.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", true, fmt.Errorf("decrypt credential: %w", err)
	}
	return string(decrypted), true, nil
}

func AuctionCredentialAAD(userID uint, field string) []byte {
	return []byte(fmt.Sprintf("auction-credential:%d:%s", userID, field))
}

func parseCredentialEncryptionKey(rawKey string) ([]byte, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, fmt.Errorf("credential encryption key is empty")
	}
	for _, decoder := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if decoded, err := decoder.DecodeString(rawKey); err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	if len(rawKey) == 32 {
		return []byte(rawKey), nil
	}
	return nil, fmt.Errorf("credential encryption key must be 32 bytes or base64-encoded 32 bytes")
}
