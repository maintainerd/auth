package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// GenerateOTP generates a random numeric OTP of the given length.
var GenerateOTP = generateOTP

func generateOTP(length int) (string, error) {
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
