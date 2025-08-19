package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	jwt.RegisteredClaims
}


func GetSignedKey(id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			Subject: id,
			Issuer: "auth.justfossa.lol",
			Audience: []string{"auth-api"},
		},
	})
	return token.SignedString(os.Getenv("JWT_SECRET"))
}
