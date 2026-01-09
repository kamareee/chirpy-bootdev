package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	if token == "" {
		t.Fatal("MakeJWT returned empty token")
	}
}

func TestValidateJWT_ValidToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	// Create a valid token
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	// Validate the token
	parsedUserID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}

	if parsedUserID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, parsedUserID)
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := -time.Hour // Token expired 1 hour ago

	// Create an expired token
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	// Try to validate the expired token
	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatal("ValidateJWT should have returned error for expired token")
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	wrongSecret := "wrong-secret"
	expiresIn := time.Hour

	// Create a token with one secret
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	// Try to validate with a different secret
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatal("ValidateJWT should have returned error for wrong secret")
	}
}

func TestValidateJWT_MalformedToken(t *testing.T) {
	tokenSecret := "test-secret"
	malformedToken := "not.a.valid.jwt"

	_, err := ValidateJWT(malformedToken, tokenSecret)
	if err == nil {
		t.Fatal("ValidateJWT should have returned error for malformed token")
	}
}

func TestValidateJWT_EmptyToken(t *testing.T) {
	tokenSecret := "test-secret"

	_, err := ValidateJWT("", tokenSecret)
	if err == nil {
		t.Fatal("ValidateJWT should have returned error for empty token")
	}
}

func TestMakeJWT_ShortExpiration(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Millisecond * 100

	// Create a token with short expiration
	token, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	// Wait for token to expire
	time.Sleep(time.Millisecond * 200)

	// Try to validate the expired token
	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatal("ValidateJWT should have returned error for expired token")
	}
}
