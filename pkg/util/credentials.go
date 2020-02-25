package util

import (
	"github.com/Peripli/service-manager/pkg/util"
	"golang.org/x/crypto/bcrypt"
)

//GenerateBrokerPlatformCredentials generates broker platform credentials
func GenerateBrokerPlatformCredentials() (string, string, string, error) {
	username, err := util.GenerateCredential()
	if err != nil {
		return "", "", "", err
	}

	password, err := util.GenerateCredential()
	if err != nil {
		return "", "", "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}

	return username, password, string(passwordHash), nil
}
