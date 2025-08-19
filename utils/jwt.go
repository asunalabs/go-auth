package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	Subject uint `json:"sub"`
	jwt.RegisteredClaims
}


func GetSignedKey(id uint) (string, string, error) {
	jti := uuid.New()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		Subject: id,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			Issuer: "auth.justfossa.lol",
			Audience: []string{"auth-api"},
			ID: jti.String(),
		},
	})

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))


	return jti.String(), t, err
}
