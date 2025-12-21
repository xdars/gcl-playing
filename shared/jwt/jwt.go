package jwt

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret []byte
	once      sync.Once
)

func getSecret() []byte {
	once.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			fmt.Println("WARNING: JWT_SECRET not set. Using insecure default. Set JWT_SECRET env var in production.")
			secret = "dev-secret-do-not-use-in-production"
		}
		jwtSecret = []byte(secret)
	})
	return jwtSecret
}

func ParseJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return getSecret(), nil
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
	return token.SignedString(getSecret())
}