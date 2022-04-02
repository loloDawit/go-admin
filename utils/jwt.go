package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
)

var SecretKey = ""

func GenerateJWT(issuer string) (string, error) {
	clamis := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    issuer,
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), //24hr
	})

	if len(SecretKey) > 0 {
		return clamis.SignedString([]byte(SecretKey))
	}

	return "", nil

}

type Clamis struct {
	jwt.StandardClaims
}

func ParseJWT(cookie string) (string, error) {
	token, err := jwt.ParseWithClaims(cookie, &Clamis{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	claims := token.Claims.(*Clamis)

	return claims.Issuer, nil

}
