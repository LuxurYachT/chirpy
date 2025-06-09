package auth

import (
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"errors"
	"time"
	"strings"
	"net/http"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			return "", err
		}
	return string(hash), nil
}

func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err
}

func MakeJWT(user_id uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	chirpyKey := []byte(tokenSecret)

	claims := jwt.RegisteredClaims{
		Issuer:		"chirpy",
		IssuedAt:	jwt.NewNumericDate(time.Now()),
		ExpiresAt:  jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:	user_id.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(chirpyKey)
	return ss, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	type MyCustomClaims struct {
		jwt.RegisteredClaims
	}

	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*MyCustomClaims)
	if !ok || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authString := headers.Get("Authorization")
	if len(authString) <= 0 {
		return "", errors.New("No token present")
	}
	tokenString := strings.TrimPrefix(authString, "Bearer ")
	return tokenString, nil
}