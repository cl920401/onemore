package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// Cipher encrypts stored upstream credentials with AES-256-GCM.
type Cipher struct {
	aead cipher.AEAD
}

// New derives a 32-byte key from the configured master key. An empty master key
// selects an insecure development key so the MVP can start without setup; never
// use that fallback in production.
func New(masterKey string) (*Cipher, error) {
	if masterKey == "" {
		masterKey = "onemore-development-master-key"
	}
	key := sha256.Sum256([]byte(masterKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plain string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}
	return c.aead.Seal(nonce, nonce, []byte(plain), nil), nil
}

func (c *Cipher) Decrypt(ciphertext []byte) (string, error) {
	size := c.aead.NonceSize()
	if len(ciphertext) < size {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, body := ciphertext[:size], ciphertext[size:]
	plain, err := c.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}
