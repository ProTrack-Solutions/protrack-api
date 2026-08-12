package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidKeySize é retornado quando a chave de criptografia não tem 32 bytes
// (AES-256 exige chave de 256 bits).
var ErrInvalidKeySize = errors.New("chave de criptografia inválida: esperado 32 bytes (base64)")

// CertCipher encapsula a criptografia/descriptografia AES-256-GCM usada para
// proteger o certificado digital A1 (dados do certificado e senha) armazenados
// em company_certificates.
type CertCipher struct {
	gcm cipher.AEAD
}

// NewCertCipher cria um CertCipher a partir de uma chave em base64 (cfg.CertEncryptionKey).
// A chave decodificada precisa ter exatamente 32 bytes.
func NewCertCipher(base64Key string) (*CertCipher, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("crypto: falha ao decodificar chave em base64: %w", err)
	}

	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: falha ao criar cipher AES: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: falha ao criar GCM: %w", err)
	}

	return &CertCipher{gcm: gcm}, nil
}

// Encrypt criptografa plaintext com AES-256-GCM, retornando o ciphertext e o
// nonce usados. O nonce é gerado aleatoriamente a cada chamada e deve ser
// persistido junto ao ciphertext para permitir a descriptografia posterior.
func (c *CertCipher) Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, err error) {
	nonce = make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("crypto: falha ao gerar nonce: %w", err)
	}

	ciphertext = c.gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt reverte Encrypt, retornando o plaintext original. Retorna erro se a
// chave estiver incorreta ou se o ciphertext/nonce tiver sido adulterado.
func (c *CertCipher) Decrypt(ciphertext []byte, nonce []byte) ([]byte, error) {
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: falha ao descriptografar (chave incorreta ou dados adulterados): %w", err)
	}

	return plaintext, nil
}
