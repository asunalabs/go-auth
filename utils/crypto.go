package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}

	return string(hash), nil
}

func ComparePassword(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashTokenSHA256(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateRefreshToken() (token string, hash string) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", ""
	}

	t := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)

	return t, HashTokenSHA256(t)
}

func CompareTokens(token string, hash string) bool {
	return HashTokenSHA256(token) == hash
}

// GenerateSecureToken generates a cryptographically secure random token
// for password resets. Returns the token and its SHA256 hash.
func GenerateSecureToken() (token string, hash string) {
	bytes := make([]byte, 32) // 256 bits
	_, err := rand.Read(bytes)
	if err != nil {
		return "", ""
	}

	token = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	hash = HashTokenSHA256(token)
	return token, hash
}
