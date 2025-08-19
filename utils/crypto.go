package utils

import (
	"crypto/rand"
	"crypto/sha256"
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
	hash :=  sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateRefreshToken() (token string, hash string) {
	t := rand.Text()
	return t, HashTokenSHA256(t)
}

func CompareTokens(token string, hash string) bool {
	return HashTokenSHA256(token) == hash
}
