package enrollment

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const agentTokenPrefix = "cfa"

var ErrInvalidAgentToken = errors.New("invalid agent token")

// GeneratePairingToken returns a UUID v4 suitable for a short-lived, single-use enrollment URL.
func GeneratePairingToken() (string, error) { return GenerateUUID() }

func GenerateSecret() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func GenerateCredential() (credentialID, secret, token string, err error) {
	credentialID, err = GenerateUUID()
	if err != nil {
		return "", "", "", err
	}
	secret, err = GenerateSecret()
	if err != nil {
		return "", "", "", err
	}
	return credentialID, secret, BuildAgentToken(credentialID, secret), nil
}

func GenerateUUID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buffer[0:4], buffer[4:6], buffer[6:8], buffer[8:10], buffer[10:16]), nil
}

func BuildAgentToken(credentialID, secret string) string {
	return agentTokenPrefix + "_" + credentialID + "_" + secret
}

func ParseAgentToken(token string) (credentialID, secret string, err error) {
	parts := strings.SplitN(token, "_", 3)
	if len(parts) != 3 || parts[0] != agentTokenPrefix || !LooksLikeUUID(parts[1]) || strings.TrimSpace(parts[2]) == "" {
		return "", "", ErrInvalidAgentToken
	}
	return parts[1], parts[2], nil
}

func LooksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		switch index {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}
