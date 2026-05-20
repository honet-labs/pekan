package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Cipher provides AES-256-GCM encryption for data at rest.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher creates a new Cipher. It hashes the provided secret using SHA-256
// to ensure a 32-byte key is used for AES-256.
func NewCipher(secret string) (*Cipher, error) {
	if secret == "" {
		return nil, errors.New("encryption secret cannot be empty")
	}
	hash := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt encrypts a plaintext string.
// It returns a base64 encoded string containing the nonce and ciphertext.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // don't encrypt empty strings
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded string containing the nonce and ciphertext.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	nonceSize := c.aead.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptDeterministic encrypts a plaintext string such that the same plaintext
// always produces the same ciphertext. This is crucial for fields that need to be
// searchable in the database (e.g., querying by phone number).
func (c *Cipher) EncryptDeterministic(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	// Use SHA-256 of the plaintext to generate a deterministic nonce
	hash := sha256.Sum256([]byte(plaintext))
	nonce := hash[:c.aead.NonceSize()]
	
	ciphertext := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}
