package crypto_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func generateBase64Key(t *testing.T) string {
	t.Helper()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("falha ao gerar chave aleatória: %v", err)
	}

	return base64.StdEncoding.EncodeToString(key)
}

// ---------------------------------------------------------------------------
// NewCertCipher Tests
// ---------------------------------------------------------------------------

func TestNewCertCipher_ValidKey(t *testing.T) {
	key := generateBase64Key(t)

	c, err := crypto.NewCertCipher(key)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve erro: %v", err)
	}
	if c == nil {
		t.Fatal("esperava CertCipher não nulo")
	}
}

func TestNewCertCipher_InvalidBase64(t *testing.T) {
	_, err := crypto.NewCertCipher("isso não é base64 válido!!!")
	if err == nil {
		t.Fatal("esperava erro para base64 inválido, obteve nil")
	}
}

func TestNewCertCipher_WrongKeySize(t *testing.T) {
	// 16 bytes (AES-128) em vez dos 32 bytes exigidos (AES-256)
	shortKey := make([]byte, 16)
	encoded := base64.StdEncoding.EncodeToString(shortKey)

	_, err := crypto.NewCertCipher(encoded)
	if err == nil {
		t.Fatal("esperava erro para chave de tamanho incorreto, obteve nil")
	}
}

// ---------------------------------------------------------------------------
// Encrypt / Decrypt round-trip Tests
// ---------------------------------------------------------------------------

func TestCertCipher_EncryptDecrypt_RoundTrip(t *testing.T) {
	key := generateBase64Key(t)
	c, err := crypto.NewCertCipher(key)
	if err != nil {
		t.Fatalf("falha ao criar CertCipher: %v", err)
	}

	plaintext := []byte("conteúdo do certificado A1 em bytes")

	ciphertext, nonce, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("falha ao criptografar: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("ciphertext não deveria ser vazio")
	}
	if len(nonce) == 0 {
		t.Fatal("nonce não deveria ser vazio")
	}

	decrypted, err := c.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("falha ao descriptografar: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("esperava plaintext '%s', obteve '%s'", plaintext, decrypted)
	}
}

func TestCertCipher_Decrypt_WrongKey(t *testing.T) {
	keyA := generateBase64Key(t)
	keyB := generateBase64Key(t)

	cipherA, err := crypto.NewCertCipher(keyA)
	if err != nil {
		t.Fatalf("falha ao criar CertCipher A: %v", err)
	}
	cipherB, err := crypto.NewCertCipher(keyB)
	if err != nil {
		t.Fatalf("falha ao criar CertCipher B: %v", err)
	}

	plaintext := []byte("senha do certificado")
	ciphertext, nonce, err := cipherA.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("falha ao criptografar: %v", err)
	}

	if _, err := cipherB.Decrypt(ciphertext, nonce); err == nil {
		t.Fatal("esperava erro ao descriptografar com chave errada, obteve nil")
	}
}

func TestCertCipher_Decrypt_TamperedCiphertext(t *testing.T) {
	key := generateBase64Key(t)
	c, err := crypto.NewCertCipher(key)
	if err != nil {
		t.Fatalf("falha ao criar CertCipher: %v", err)
	}

	plaintext := []byte("conteúdo original do certificado")
	ciphertext, nonce, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("falha ao criptografar: %v", err)
	}

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[0] ^= 0xFF

	if _, err := c.Decrypt(tampered, nonce); err == nil {
		t.Fatal("esperava erro ao descriptografar ciphertext adulterado, obteve nil")
	}
}

func TestCertCipher_Encrypt_DifferentNoncePerCall(t *testing.T) {
	key := generateBase64Key(t)
	c, err := crypto.NewCertCipher(key)
	if err != nil {
		t.Fatalf("falha ao criar CertCipher: %v", err)
	}

	plaintext := []byte("mesmo conteúdo, chamadas diferentes")

	_, nonce1, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("falha ao criptografar (1): %v", err)
	}
	_, nonce2, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("falha ao criptografar (2): %v", err)
	}

	if string(nonce1) == string(nonce2) {
		t.Error("esperava nonces diferentes entre chamadas, obteve nonces iguais")
	}
}
