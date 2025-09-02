package auth

import (
	// "fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestGetBearerToken(t *testing.T) {
	myHeadersMap := map[string]string{
		"Content-Type": "application/json",
		"Authorization": "Bearer some_token_value",
		"X-Custom-Header": "MyValue",
	}

	headers := make(http.Header)

	for key, value := range myHeadersMap {
		headers.Set(key, value) 
	}

	token, err := GetBearerToken(headers)
	if err != nil || token != "some_token_value" {
		t.Errorf(`GetBearerToken should return some_token_value, nil, not %q, %v`, token, err)
	}
}

func TestHashPassword(t *testing.T) {
	password := "123456"
	hashed, err := HashPassword(password)
	if bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) != nil || err != nil {
		t.Errorf(`HashPassword(%q) = %q, %v`, password, hashed, err)
	}
}

func TestJWT(t *testing.T) {
	tokenSecret := "123456"
	userId, err := uuid.Parse("be93db0d-4c6d-49cf-b56d-ba22392eb160")
	if err != nil {
		t.Errorf(`err should be nil, err: %v`, err)
	}
	expiresIn := 1 * time.Second
	tokenString, err := MakeJWT(userId, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf(`err should be nil, err: %v`, err)
	}
	
	valUUID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil || userId != valUUID {
		t.Errorf(`ValidateJWT(%q, %q) = %q, %v, want %v, nil`, tokenString, tokenSecret, valUUID, err, userId)
	}
}

func TestJWTExpired(t *testing.T) {
	tokenSecret := "123456"
	userId, err := uuid.Parse("be93db0d-4c6d-49cf-b56d-ba22392eb160")
	if err != nil {
		t.Errorf(`err should be nil, err: %v`, err)
	}
	expiresIn := 1 * time.Second
	tokenString, err := MakeJWT(userId, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf(`err should be nil, err: %v`, err)
	}

	time.Sleep(2* time.Second)
	valUUID, err := ValidateJWT(tokenString, tokenSecret)
	if err == nil || userId == valUUID {
		t.Errorf(`ValidateJWT(%q, %q) = %q, %v, want nil, not nil`, tokenString, tokenSecret, valUUID, err)
	}	
}