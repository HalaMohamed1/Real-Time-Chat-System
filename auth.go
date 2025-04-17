// This file is used to create and validate the jwt token

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Secret key for signing tokens
var secretKey = []byte("my-super-secret-key-123!@#$%^&*")

// generates a new JWT token for a user
func CreateToken(username string) (string, error) {
	// Create a new token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		})

	// Sign the token with our secret key
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}

	return tokenString, nil
}

// checks if a token is valid and returns the claims
func ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	// Check if token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Get the claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("error getting claims")
	}

	return &claims, nil
}
