package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 🔐 Секретный ключ (пока жёстко прописан — потом вынеси в .env)
var jwtKey = []byte("super_secret_key")

// Claims — структура для JWT (похожа на пример из Lab-4)
type Claims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT создаёт токен
func GenerateJWT(userID int, role string) (string, error) {
	fmt.Printf("DEBUG: GenerateJWT called for userID=%d role=%s\n", userID, role)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtKey)
	if err != nil {
		fmt.Printf("DEBUG: GenerateJWT signing error: %v\n", err)
		return "", err
	}
	fmt.Printf("DEBUG: GenerateJWT success tokenLen=%d\n", len(tokenStr))
	return tokenStr, nil
}

// ParseJWT проверяет и возвращает Claims
func ParseJWT(tokenStr string) (*Claims, error) {
	fmt.Printf("DEBUG: ParseJWT called tokenLen=%d\n", len(tokenStr))
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		fmt.Printf("DEBUG: ParseJWT parse error: %v\n", err)
		return nil, err
	}
	if !token.Valid {
		fmt.Printf("DEBUG: ParseJWT invalid token\n")
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
