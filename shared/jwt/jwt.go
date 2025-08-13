package jwt

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var jwtSecret = []byte("secret-key")

func ParseJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims["sub"].(string), nil
	} else {
		return "", fmt.Errorf("invalid token")
	}
}

func GenerateJWT(email string) (string, error) {
	claims := jwt.MapClaims{
		"sub": email,
		"exp": time.Now().Add(time.Hour * 1).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}