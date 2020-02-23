package util

import (
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

//GenerateBrokerPlatformCredentials generates broker platform credentials
func GenerateBrokerPlatformCredentials() (string, string, string, error) {
	username, err := generateCredential()
	if err != nil {
		return "", "", "", err
	}

	password, err := generateCredential()
	if err != nil {
		return "", "", "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}

	return username, password, string(passwordHash), nil
}

func generateCredential() (string, error) {
	newCredential := make([]byte, 32)
	if _, err := rand.Read(newCredential); err != nil {
		return "", fmt.Errorf("could not generate credential: %v", err)
	}

	return string(newCredential), nil
}
