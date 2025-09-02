package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func GetBearerToken(headers http.Header) (string, error) {
	authorization := headers.Get("Authorization")
	if authorization == "" {
		return "", errors.New("no header Authorization")
	}
	token, found := strings.CutPrefix(authorization, "Bearer ")
	if !found {
		return "", errors.New("not found format Bearer")
	}
	return token, nil
}

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func CheckPasswordHash(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return err
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  &jwt.NumericDate{time.Now()},
			ExpiresAt: &jwt.NumericDate{time.Now().Add(expiresIn)},
			Subject:   userID.String(),
		},
	)
	fmt.Printf("jwt.NumericDate{time.Now().Add(expiresIn)}: %v\n", jwt.NumericDate{time.Now().Add(expiresIn)})
	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimStruct := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		&claimStruct,
		func(t *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.Nil, err
	}
	uuid_value, err := uuid.Parse(claimStruct.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid_value, nil
}
