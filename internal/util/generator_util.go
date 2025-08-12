package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
)

// GenerateIdentifier returns a random URL-safe identifier string
func GenerateIdentifier(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // or return empty string + error
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

// GenerateOTP generates a random integer OTP between 0 and max (inclusive).
func GenerateOTP(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}

	// Digits allowed in OTP
	const digits = "0123456789"
	var otp strings.Builder

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp.WriteByte(digits[n.Int64()])
	}

	return otp.String(), nil
}
