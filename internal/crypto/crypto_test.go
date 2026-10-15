package crypto

import (
	"testing"
)

func TestAESEncryptionDecryption(t *testing.T) {
	key, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}

	plaintext := []byte("SECRET_ENV_VALUE=super_sensitive_api_key_12345")
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected decrypted %s, got %s", string(plaintext), string(decrypted))
	}
}

func TestArgon2PasswordHash(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Fatal("expected password to match hash")
	}

	wrongOk, err := VerifyPassword("WrongPassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword with wrong password failed: %v", err)
	}
	if wrongOk {
		t.Fatal("expected wrong password to fail verification")
	}
}

func TestTokenGeneration(t *testing.T) {
	raw, hash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken failed: %v", err)
	}
	if len(raw) == 0 || len(hash) != 64 {
		t.Fatalf("unexpected token length: raw=%d, hash=%d", len(raw), len(hash))
	}
	if SHA256String(raw) != hash {
		t.Fatalf("hash mismatch")
	}
}
