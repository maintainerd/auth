package security

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PasswordStrengthScore rates a password on a 0-4 scale using a simplified
// zxcvbn-like model. 0 = extremely weak, 4 = very strong.
func PasswordStrengthScore(password string) int {
	if len(password) == 0 {
		return 0
	}

	score := 0

	passwordLength := utf8.RuneCountInString(password)
	if passwordLength >= 8 {
		score++
	}
	if passwordLength >= 12 {
		score++
	}
	if passwordLength >= 16 {
		score++
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	classCount := 0

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			if !hasUpper {
				hasUpper = true
				classCount++
			}
		case unicode.IsLower(r):
			if !hasLower {
				hasLower = true
				classCount++
			}
		case unicode.IsDigit(r):
			if !hasDigit {
				hasDigit = true
				classCount++
			}
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !hasSpecial {
				hasSpecial = true
				classCount++
			}
		}
	}

	if classCount >= 2 {
		score++
	}
	if classCount >= 3 {
		score++
	}
	if classCount >= 4 {
		score++
	}

	if hasRepeatingPatterns(password) {
		score = max(0, score-2)
	}
	if isCommonPattern(password) {
		score = max(0, score-1)
	}

	return clampScore(score, 4)
}

func hasRepeatingPatterns(s string) bool {
	lower := strings.ToLower(s)
	common := []string{"123456", "qwerty", "asdfgh", "zxcvbn", "abc", "aaa", "111"}
	for _, p := range common {
		if strings.Contains(lower, p) {
			return true
		}
	}
	for i := 0; i < len(s)-2; i++ {
		if s[i] == s[i+1] && s[i+1] == s[i+2] {
			return true
		}
	}
	return false
}

func isCommonPattern(s string) bool {
	lower := strings.ToLower(s)
	patterns := []string{"password", "admin", "test", "abcd", "qwerty", "iloveyou", "monkey"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func clampScore(score, maxScore int) int {
	if score < 0 {
		return 0
	}
	return min(score, maxScore)
}
